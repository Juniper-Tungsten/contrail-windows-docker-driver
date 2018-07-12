//
// Copyright (c) 2018 Juniper Networks, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Implemented according to
// https://github.com/docker/libnetwork/blob/master/docs/remote.md

package cnm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"context"

	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	winio "github.com/Microsoft/go-winio"
	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/netlabel"
	log "github.com/sirupsen/logrus"
)

type ServerCNM struct {
	// TODO: for now, Core field is public, because we need to access its fields, like controller.
	// This should be made private when making the Controller port smaller.
	Core *driver_core.ContrailDriverCore
	// TODO: we need to keep the following fields for now, but the plan is to refactor them
	// out (along with related pipe logic) to a separate primary adapter.
	listener           net.Listener
	PipeAddr           string
	stopReasonChan     chan error
	stoppedServingChan chan interface{}
	IsServing          bool
}

type NetworkMeta struct {
	tenant     string
	network    string
	subnetCIDR string
}

func NewServerCNM(core *driver_core.ContrailDriverCore) *ServerCNM {
	d := &ServerCNM{
		Core:               core,
		PipeAddr:           "//./pipe/" + common.DriverName,
		stopReasonChan:     make(chan error, 1),
		stoppedServingChan: make(chan interface{}, 1),
		IsServing:          false,
	}
	return d
}

func (d *ServerCNM) StartServing() error {

	if d.IsServing {
		return errors.New("Already serving.")
	}

	startedServingChan := make(chan interface{}, 1)
	failedChan := make(chan error, 1)

	go func() {

		defer func() {
			d.IsServing = false
			d.stoppedServingChan <- true
		}()

		pipeConfig := winio.PipeConfig{
			// This will set permissions for Service, System, Adminstrator group and account to
			// have full access
			SecurityDescriptor: "D:(A;ID;FA;;;SY)(A;ID;FA;;;BA)(A;ID;FA;;;LA)(A;ID;FA;;;LS)",
			MessageMode:        true,
			InputBufferSize:    4096,
			OutputBufferSize:   4096,
		}

		var err error
		d.listener, err = winio.ListenPipe(d.PipeAddr, &pipeConfig)
		if err != nil {
			failedChan <- errors.New(fmt.Sprintln("When setting up listener:", err))
			return
		}

		if err := d.waitForPipeToAppear(); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When waiting for pipe to appear:", err))
			return
		}

		h := network.NewHandler(d)
		go func() {
			err := h.Serve(d.listener)
			if err != nil {
				d.stopReasonChan <- errors.New(fmt.Sprintln("When serving:", err))
			}
		}()

		if err := d.waitUntilPipeDialable(); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When waiting for pipe to be dialable:", err))
			return
		}

		if err := os.MkdirAll(common.PluginSpecDir(), 0755); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When setting up plugin spec directory:", err))
			return
		}

		url := "npipe://" + d.listener.Addr().String()
		if err := ioutil.WriteFile(common.PluginSpecFilePath(), []byte(url), 0644); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When creating spec file:", err))
			return
		}

		d.IsServing = true
		startedServingChan <- true

		if err := <-d.stopReasonChan; err != nil {
			log.Errorln("Stopped serving because:", err)
		}

		log.Infoln("Closing npipe listener")
		if err := d.listener.Close(); err != nil {
			log.Warnln("When closing listener:", err)
		}

		log.Infoln("Removing spec file")
		if err := os.Remove(common.PluginSpecFilePath()); err != nil {
			log.Warnln("When removing spec file:", err)
		}

		if err := d.waitForPipeToStop(); err != nil {
			log.Warnln("Failed to properly close named pipe, but will continue anyways:", err)
		}
	}()

	select {
	case <-startedServingChan:
		log.Infoln("Started serving on ", d.PipeAddr)
		return nil
	case err := <-failedChan:
		log.Error(err)
		return err
	}
}

func (d *ServerCNM) StopServing() error {
	if d.IsServing {
		d.stopReasonChan <- nil
		<-d.stoppedServingChan
		log.Infoln("Stopped serving")
	}

	return nil
}

func (d *ServerCNM) GetCapabilities() (*network.CapabilitiesResponse, error) {
	log.Debugln("=== GetCapabilities")
	r := &network.CapabilitiesResponse{}
	r.Scope = network.LocalScope
	return r, nil
}

func (d *ServerCNM) CreateNetwork(req *network.CreateNetworkRequest) error {
	log.Debugln("=== CreateNetwork")
	log.Debugln("network.NetworkID =", req.NetworkID)
	log.Debugln(req)
	log.Debugln("IPv4:")
	for _, n := range req.IPv4Data {
		log.Debugln(n)
	}
	log.Debugln("IPv6:")
	for _, n := range req.IPv6Data {
		log.Debugln(n)
	}
	log.Debugln("options:")
	for k, v := range req.Options {
		fmt.Printf("%v: %v\n", k, v)
	}

	reqGenericOptionsMap, exists := req.Options[netlabel.GenericData]
	if !exists {
		return errors.New("Generic options missing")
	}

	genericOptions, ok := reqGenericOptionsMap.(map[string]interface{})
	if !ok {
		return errors.New("Malformed generic options")
	}

	tenant, exists := genericOptions["tenant"]
	if !exists {
		return errors.New("Tenant not specified")
	}

	netName, exists := genericOptions["network"]
	if !exists {
		return errors.New("Network name not specified")
	}

	// this is subnet already in CIDR format
	if len(req.IPv4Data) == 0 {
		return errors.New("Docker subnet IPv4 data missing")
	}
	subnetCIDR := req.IPv4Data[0].Pool

	tenantName := tenant.(string)
	networkName := netName.(string)

	return d.Core.CreateNetwork(req.NetworkID, tenantName, networkName, subnetCIDR)
}

func (d *ServerCNM) AllocateNetwork(req *network.AllocateNetworkRequest) (
	*network.AllocateNetworkResponse, error) {
	log.Debugln("=== AllocateNetwork")
	log.Debugln(req)
	// This method is used in swarm, in remote plugins. We don't implement it.
	return nil, errors.New("AllocateNetwork is not implemented")
}

func (d *ServerCNM) DeleteNetwork(req *network.DeleteNetworkRequest) error {
	log.Debugln("=== DeleteNetwork")
	log.Debugln(req)

	return d.Core.DeleteNetwork(req.NetworkID)
}

func (d *ServerCNM) FreeNetwork(req *network.FreeNetworkRequest) error {
	log.Debugln("=== FreeNetwork")
	log.Debugln(req)
	// This method is used in swarm, in remote plugins. We don't implement it.
	return errors.New("FreeNetwork is not implemented")
}

func (d *ServerCNM) CreateEndpoint(req *network.CreateEndpointRequest) (
	*network.CreateEndpointResponse, error) {
	log.Debugln("=== CreateEndpoint")
	log.Debugln(req)
	log.Debugln(req.Interface)
	log.Debugln(req.EndpointID)
	log.Debugln("options:")
	for k, v := range req.Options {
		fmt.Printf("%v: %v\n", k, v)
	}

	container, err := d.Core.CreateEndpoint(req.NetworkID, req.EndpointID)
	if err != nil {
		return nil, err
	}

	ipCIDR := fmt.Sprintf("%s/%v", container.IP, container.PrefixLen)
	r := &network.CreateEndpointResponse{
		Interface: &network.EndpointInterface{
			Address:    ipCIDR,
			MacAddress: container.Mac,
		},
	}
	return r, nil
}

func (d *ServerCNM) DeleteEndpoint(req *network.DeleteEndpointRequest) error {
	log.Debugln("=== DeleteEndpoint")
	log.Debugln(req)

	// TODO JW-187.
	// We need something like:
	// containerID := req.Options["vmname"]
	containerID := req.EndpointID

	// TODO: remove this function and call, so that we don't have to rely on network meta from
	// docker. We want pure CNM plugin, not one that relies on docker running on local host.
	meta, err := d.networkMetaFromDockerNetwork(req.NetworkID)
	if err != nil {
		return err
	}

	contrailNetwork, err := d.Core.Controller.GetNetwork(meta.tenant, meta.network)
	if err != nil {
		return err
	}
	log.Infoln("Retrieved Contrail network:", contrailNetwork.GetUuid())

	contrailVif, err := d.Core.Controller.GetExistingInterface(contrailNetwork, meta.tenant,
		containerID)
	if err != nil {
		log.Warn("When handling DeleteEndpoint, interface wasn't found")
	} else {
		go func() {
			err := d.Core.Agent.DeletePort(contrailVif.GetUuid())
			if err != nil {
				log.Error(err.Error())
			}
		}()
	}

	contrailInstance, err := d.Core.Controller.GetInstance(containerID)
	if err != nil {
		log.Warn("When handling DeleteEndpoint, Contrail vm instance wasn't found")
	} else {
		err = d.Core.Controller.DeleteElementRecursive(contrailInstance)
		if err != nil {
			log.Warn("When handling DeleteEndpoint, failed to remove Contrail vm instance")
		}
	}

	hnsEpName := req.EndpointID
	epToDelete, err := d.Core.LocalContrailEndpointsRepo.GetEndpointByName(hnsEpName)
	if err != nil {
		return err
	}
	if epToDelete == nil {
		log.Warn("When handling DeleteEndpoint, couldn't find HNS endpoint to delete")
		return nil
	}

	return d.Core.LocalContrailEndpointsRepo.DeleteEndpoint(epToDelete.Id)
}

func (d *ServerCNM) EndpointInfo(req *network.InfoRequest) (*network.InfoResponse, error) {
	log.Debugln("=== EndpointInfo")
	log.Debugln(req)

	hnsEpName := req.EndpointID
	hnsEp, err := d.Core.LocalContrailEndpointsRepo.GetEndpointByName(hnsEpName)
	if err != nil {
		return nil, err
	}
	if hnsEp == nil {
		return nil, errors.New("When handling EndpointInfo, couldn't find HNS endpoint")
	}

	respData := map[string]string{
		"hnsid":             hnsEp.Id,
		netlabel.MacAddress: hnsEp.MacAddress,
	}

	r := &network.InfoResponse{
		Value: respData,
	}
	return r, nil
}

func (d *ServerCNM) Join(req *network.JoinRequest) (*network.JoinResponse, error) {
	log.Debugln("=== Join")
	log.Debugln(req)
	log.Debugln("options:")
	for k, v := range req.Options {
		fmt.Printf("%v: %v\n", k, v)
	}

	hnsEp, err := d.Core.LocalContrailEndpointsRepo.GetEndpointByName(req.EndpointID)
	if err != nil {
		return nil, err
	}
	if hnsEp == nil {
		return nil, errors.New("Such HNS endpoint doesn't exist")
	}

	r := &network.JoinResponse{
		DisableGatewayService: true,
		Gateway:               hnsEp.GatewayAddress,
	}

	return r, nil
}

func (d *ServerCNM) Leave(req *network.LeaveRequest) error {
	log.Debugln("=== Leave")
	log.Debugln(req)

	hnsEp, err := d.Core.LocalContrailEndpointsRepo.GetEndpointByName(req.EndpointID)
	if err != nil {
		return err
	}
	if hnsEp == nil {
		return errors.New("Such HNS endpoint doesn't exist")
	}

	return nil
}

func (d *ServerCNM) DiscoverNew(req *network.DiscoveryNotification) error {
	log.Debugln("=== DiscoverNew")
	log.Debugln(req)
	// We don't care about discovery notifications.
	return nil
}

func (d *ServerCNM) DiscoverDelete(req *network.DiscoveryNotification) error {
	log.Debugln("=== DiscoverDelete")
	log.Debugln(req)
	// We don't care about discovery notifications.
	return nil
}

func (d *ServerCNM) ProgramExternalConnectivity(
	req *network.ProgramExternalConnectivityRequest) error {
	log.Debugln("=== ProgramExternalConnectivity")
	log.Debugln(req)
	return nil
}

func (d *ServerCNM) RevokeExternalConnectivity(
	req *network.RevokeExternalConnectivityRequest) error {
	log.Debugln("=== RevokeExternalConnectivity")
	log.Debugln(req)
	return nil
}

func (d *ServerCNM) waitForPipeToAppear() error {
	return d.waitForPipe(true)
}

func (d *ServerCNM) waitForPipeToStop() error {
	return d.waitForPipe(false)
}

func (d *ServerCNM) waitForPipe(waitUntilExists bool) error {
	timeStarted := time.Now()
	for {
		if time.Since(timeStarted) > common.PipePollingTimeout {
			return errors.New("Waited for pipe file for too long.")
		}

		_, err := os.Stat(d.PipeAddr)

		// if waitUntilExists is true, we wait for the file to appear in filesystem.
		// else, we wait for the file to disappear from the filesystem.
		if fileExists := !os.IsNotExist(err); fileExists == waitUntilExists {
			break
		} else {
			log.Errorf("Waiting for pipe file, but: %s", err)
		}

		time.Sleep(common.PipePollingRate)
	}

	return nil
}

func (d *ServerCNM) waitUntilPipeDialable() error {
	timeStarted := time.Now()
	for {
		if time.Since(timeStarted) > common.PipePollingTimeout {
			return errors.New("Waited for pipe to be dialable for too long.")
		}

		timeout := time.Millisecond * 10
		conn, err := sockets.DialPipe(d.PipeAddr, timeout)
		if err == nil {
			conn.Close()
			return nil
		}

		log.Errorf("Waiting until dialable, but: %s", err)

		time.Sleep(common.PipePollingRate)
	}
}

func (d *ServerCNM) networkMetaFromDockerNetwork(dockerNetID string) (*NetworkMeta,
	error) {
	docker, err := dockerClient.NewEnvClient()
	if err != nil {
		return nil, err
	}

	inspectOptions := dockerTypes.NetworkInspectOptions{
		Scope:   "",
		Verbose: false,
	}
	dockerNetwork, err := docker.NetworkInspect(context.Background(), dockerNetID, inspectOptions)
	if err != nil {
		return nil, err
	}

	var meta NetworkMeta
	var exists bool

	meta.tenant, exists = dockerNetwork.Options["tenant"]
	if !exists {
		return nil, errors.New("Retrieved network has no Contrail tenant specified")
	}

	meta.network, exists = dockerNetwork.Options["network"]
	if !exists {
		return nil, errors.New("Retrieved network has no Contrail network name specfied")
	}

	ipamCfg := dockerNetwork.IPAM.Config
	if len(ipamCfg) == 0 {
		return nil, errors.New("No configured subnets in docker network")
	}
	meta.subnetCIDR = ipamCfg[0].Subnet

	return &meta, nil
}
