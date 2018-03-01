//
// Copyright (c) 2017 Juniper Networks, Inc. All Rights Reserved.
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

package driver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"context"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/agent"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/controller"
	"github.com/Juniper/contrail-windows-docker-driver/hns"
	"github.com/Juniper/contrail-windows-docker-driver/hnsManager"
	"github.com/Juniper/contrail-windows-docker-driver/hyperv"
	"github.com/Microsoft/go-winio"
	"github.com/Microsoft/hcsshim"
	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/netlabel"
	log "github.com/sirupsen/logrus"
)

type ContrailDriver struct {
	controller         *controller.Controller
	agent              agent.Agent
	hnsMgr             *hnsManager.HNSManager
	networkAdapter     common.AdapterName
	vswitchName        common.VSwitchName
	listener           net.Listener
	PipeAddr           string
	stopChan           chan interface{}
	stoppedServingChan chan interface{}
	IsServing          bool
}

type NetworkMeta struct {
	tenant     string
	network    string
	subnetCIDR string
}

func NewDriver(adapter, vswitchName string, c *controller.Controller, agent agent.Agent) *ContrailDriver {

	d := &ContrailDriver{
		controller:         c,
		agent:              agent,
		hnsMgr:             &hnsManager.HNSManager{},
		networkAdapter:     common.AdapterName(adapter),
		vswitchName:        common.VSwitchName(vswitchName),
		PipeAddr:           "//./pipe/" + common.DriverName,
		stopChan:           make(chan interface{}, 1),
		stoppedServingChan: make(chan interface{}, 1),
		IsServing:          false,
	}
	return d
}

func (d *ContrailDriver) StartServing() error {

	if d.IsServing {
		return errors.New("Already serving.")
	}

	if err := d.createRootNetwork(); err != nil {
		return err
	}

	running, err := hyperv.IsExtensionRunning(d.vswitchName)
	if err != nil {
		return err
	}

	if !running {
		return errors.New("Extension doesn't seem to be running. Maybe try reinstalling?")
	}

	enabled, err := hyperv.IsExtensionEnabled(d.vswitchName)
	if err != nil {
		return err
	}

	if !enabled {
		if err := hyperv.EnableExtension(d.vswitchName); err != nil {
			return err
		}

		running, err := hyperv.IsExtensionRunning(d.vswitchName)
		if err != nil {
			return err
		}

		if !running {
			return errors.New("Extension stopped running after being enabled. " +
				"Try stopping vRouter agent, docker and removing container networks.")
		}
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

		h := network.NewHandler(d)
		go h.Serve(d.listener)

		if err := os.MkdirAll(common.PluginSpecDir(), 0755); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When setting up plugin spec directory:", err))
			return
		}

		url := "npipe://" + d.listener.Addr().String()
		if err := ioutil.WriteFile(common.PluginSpecFilePath(), []byte(url), 0644); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When creating spec file:", err))
			return
		}

		if err := d.waitForPipeToStart(); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When waiting for pipe to start:", err))
			return
		}

		d.IsServing = true
		startedServingChan <- true

		<-d.stopChan

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

func (d *ContrailDriver) StopServing() error {
	if d.IsServing {
		d.stopChan <- true
		<-d.stoppedServingChan
		log.Infoln("Stopped serving")
	}

	return nil
}

func (d *ContrailDriver) GetCapabilities() (*network.CapabilitiesResponse, error) {
	log.Debugln("=== GetCapabilities")
	r := &network.CapabilitiesResponse{}
	r.Scope = network.LocalScope
	return r, nil
}

func (d *ContrailDriver) CreateNetwork(req *network.CreateNetworkRequest) error {
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
	ipPool := req.IPv4Data[0].Pool

	// Check if network is already created in Contrail.
	contrailNetwork, err := d.controller.GetNetwork(tenant.(string), netName.(string))
	if err != nil {
		return err
	}
	if contrailNetwork == nil {
		return errors.New("Retrieved Contrail network is nil")
	}

	log.Infoln("Got Contrail network", contrailNetwork.GetDisplayName())

	contrailIpam, err := d.controller.GetIpamSubnet(contrailNetwork, ipPool)
	if err != nil {
		return err
	}
	subnetCIDR := d.getContrailSubnetCIDR(contrailIpam)

	contrailGateway := contrailIpam.DefaultGateway
	if contrailGateway == "" {
		return errors.New("Default GW is empty")
	}

	_, err = d.hnsMgr.CreateNetwork(d.networkAdapter, tenant.(string), netName.(string),
		subnetCIDR, contrailGateway)

	return err
}

func (d *ContrailDriver) AllocateNetwork(req *network.AllocateNetworkRequest) (
	*network.AllocateNetworkResponse, error) {
	log.Debugln("=== AllocateNetwork")
	log.Debugln(req)
	// This method is used in swarm, in remote plugins. We don't implement it.
	return nil, errors.New("AllocateNetwork is not implemented")
}

func (d *ContrailDriver) DeleteNetwork(req *network.DeleteNetworkRequest) error {
	log.Debugln("=== DeleteNetwork")
	log.Debugln(req)

	dockerNetsMeta, err := d.dockerNetworksMeta()
	log.Debugln("Current docker-Contrail networks meta", dockerNetsMeta)
	if err != nil {
		return err
	}

	hnsNetsMeta, err := d.hnsNetworksMeta()
	log.Debugln("Current HNS-Contrail networks meta", hnsNetsMeta)
	if err != nil {
		return err
	}

	var toRemove *NetworkMeta
	toRemove = nil
	for _, hnsMeta := range hnsNetsMeta {
		matchFound := false
		for _, dockerMeta := range dockerNetsMeta {
			if dockerMeta.tenant == hnsMeta.tenant && dockerMeta.network == hnsMeta.network &&
				dockerMeta.subnetCIDR == hnsMeta.subnetCIDR {
				matchFound = true
				break
			}
		}
		if !matchFound {
			toRemove = &hnsMeta
			break
		}
	}

	if toRemove == nil {
		return errors.New("During handling of DeleteNetwork, couldn't find net to remove")
	}
	return d.hnsMgr.DeleteNetwork(toRemove.tenant, toRemove.network, toRemove.subnetCIDR)
}

func (d *ContrailDriver) FreeNetwork(req *network.FreeNetworkRequest) error {
	log.Debugln("=== FreeNetwork")
	log.Debugln(req)
	// This method is used in swarm, in remote plugins. We don't implement it.
	return errors.New("FreeNetwork is not implemented")
}

func (d *ContrailDriver) CreateEndpoint(req *network.CreateEndpointRequest) (
	*network.CreateEndpointResponse, error) {
	log.Debugln("=== CreateEndpoint")
	log.Debugln(req)
	log.Debugln(req.Interface)
	log.Debugln(req.EndpointID)
	log.Debugln("options:")
	for k, v := range req.Options {
		fmt.Printf("%v: %v\n", k, v)
	}

	meta, err := d.networkMetaFromDockerNetwork(req.NetworkID)
	if err != nil {
		return nil, err
	}

	contrailNetwork, err := d.controller.GetNetwork(meta.tenant, meta.network)
	if err != nil {
		return nil, err
	}
	log.Infoln("Retrieved Contrail network:", contrailNetwork.GetUuid())

	// TODO JW-187.
	// We need to retreive Container ID here and use it instead of EndpointID as
	// argument to d.controller.GetOrCreateInstance().
	// EndpointID is equiv to interface, but in Contrail, we have a "VirtualMachine" in
	// data model.
	// A single VM can be connected to two or more overlay networks, but when we use
	// EndpointID, this won't work.
	// We need something like:
	// containerID := req.Options["vmname"]
	containerID := req.EndpointID

	contrailIpam, err := d.controller.GetIpamSubnet(contrailNetwork, meta.subnetCIDR)
	if err != nil {
		return nil, err
	}
	contrailSubnetCIDR := d.getContrailSubnetCIDR(contrailIpam)

	contrailVif, err := d.controller.GetOrCreateInterface(contrailNetwork, meta.tenant,
		containerID)
	if err != nil {
		return nil, err
	}

	contrailVM, err := d.controller.GetOrCreateInstance(contrailVif, containerID)
	if err != nil {
		return nil, err
	}

	contrailIP, err := d.controller.GetOrCreateInstanceIp(contrailNetwork, contrailVif, contrailIpam.SubnetUuid)
	if err != nil {
		return nil, err
	}
	instanceIP := contrailIP.GetInstanceIpAddress()
	log.Infoln("Retrieved instance IP:", instanceIP)

	contrailGateway := contrailIpam.DefaultGateway
	log.Infoln("Retrieved GW address:", contrailGateway)
	if contrailGateway == "" {
		return nil, errors.New("Default GW is empty")
	}

	contrailMac, err := d.controller.GetInterfaceMac(contrailVif)
	log.Infoln("Retrieved MAC:", contrailMac)
	if err != nil {
		return nil, err
	}
	// contrail MACs are like 11:22:aa:bb:cc:dd
	// HNS needs MACs like 11-22-AA-BB-CC-DD
	formattedMac := strings.Replace(strings.ToUpper(contrailMac), ":", "-", -1)

	hnsNet, err := d.hnsMgr.GetNetwork(meta.tenant, meta.network, contrailSubnetCIDR)
	if err != nil {
		return nil, err
	}

	hnsEndpointConfig := &hcsshim.HNSEndpoint{
		VirtualNetworkName: hnsNet.Name,
		Name:               req.EndpointID,
		IPAddress:          net.ParseIP(instanceIP),
		MacAddress:         formattedMac,
		GatewayAddress:     contrailGateway,
	}

	hnsEndpointID, err := hns.CreateHNSEndpoint(hnsEndpointConfig)
	if err != nil {
		return nil, err
	}

	// TODO: test this when Agent is ready
	ifName := d.generateFriendlyName(hnsEndpointID)

	go d.agent.AddPort(contrailVM.GetUuid(), contrailVif.GetUuid(), ifName, contrailMac, containerID,
		contrailIP.GetInstanceIpAddress(), contrailNetwork.GetUuid())

	epAddressCIDR := fmt.Sprintf("%s/%v", instanceIP, contrailIpam.Subnet.IpPrefixLen)
	r := &network.CreateEndpointResponse{
		Interface: &network.EndpointInterface{
			Address:    epAddressCIDR,
			MacAddress: contrailMac,
		},
	}
	return r, nil
}

func (d *ContrailDriver) DeleteEndpoint(req *network.DeleteEndpointRequest) error {
	log.Debugln("=== DeleteEndpoint")
	log.Debugln(req)

	// TODO JW-187.
	// We need something like:
	// containerID := req.Options["vmname"]
	containerID := req.EndpointID

	meta, err := d.networkMetaFromDockerNetwork(req.NetworkID)
	if err != nil {
		return err
	}

	contrailNetwork, err := d.controller.GetNetwork(meta.tenant, meta.network)
	if err != nil {
		return err
	}
	log.Infoln("Retrieved Contrail network:", contrailNetwork.GetUuid())

	contrailVif, err := d.controller.GetExistingInterface(contrailNetwork, meta.tenant,
		containerID)
	if err != nil {
		log.Warn("When handling DeleteEndpoint, interface wasn't found")
	} else {
		go d.agent.DeletePort(contrailVif.GetUuid())
	}

	contrailInstance, err := types.VirtualMachineByName(d.controller.ApiClient, containerID)
	if err != nil {
		log.Warn("When handling DeleteEndpoint, Contrail vm instance wasn't found")
	} else {
		err = d.controller.DeleteElementRecursive(contrailInstance)
		if err != nil {
			log.Warn("When handling DeleteEndpoint, failed to remove Contrail vm instance")
		}
	}

	hnsEpName := req.EndpointID
	epToDelete, err := hns.GetHNSEndpointByName(hnsEpName)
	if err != nil {
		return err
	}
	if epToDelete == nil {
		log.Warn("When handling DeleteEndpoint, couldn't find HNS endpoint to delete")
		return nil
	}

	return hns.DeleteHNSEndpoint(epToDelete.Id)
}

func (d *ContrailDriver) EndpointInfo(req *network.InfoRequest) (*network.InfoResponse, error) {
	log.Debugln("=== EndpointInfo")
	log.Debugln(req)

	hnsEpName := req.EndpointID
	hnsEp, err := hns.GetHNSEndpointByName(hnsEpName)
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

func (d *ContrailDriver) Join(req *network.JoinRequest) (*network.JoinResponse, error) {
	log.Debugln("=== Join")
	log.Debugln(req)
	log.Debugln("options:")
	for k, v := range req.Options {
		fmt.Printf("%v: %v\n", k, v)
	}

	hnsEp, err := hns.GetHNSEndpointByName(req.EndpointID)
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

func (d *ContrailDriver) Leave(req *network.LeaveRequest) error {
	log.Debugln("=== Leave")
	log.Debugln(req)

	hnsEp, err := hns.GetHNSEndpointByName(req.EndpointID)
	if err != nil {
		return err
	}
	if hnsEp == nil {
		return errors.New("Such HNS endpoint doesn't exist")
	}

	return nil
}

func (d *ContrailDriver) DiscoverNew(req *network.DiscoveryNotification) error {
	log.Debugln("=== DiscoverNew")
	log.Debugln(req)
	// We don't care about discovery notifications.
	return nil
}

func (d *ContrailDriver) DiscoverDelete(req *network.DiscoveryNotification) error {
	log.Debugln("=== DiscoverDelete")
	log.Debugln(req)
	// We don't care about discovery notifications.
	return nil
}

func (d *ContrailDriver) ProgramExternalConnectivity(
	req *network.ProgramExternalConnectivityRequest) error {
	log.Debugln("=== ProgramExternalConnectivity")
	log.Debugln(req)
	return nil
}

func (d *ContrailDriver) RevokeExternalConnectivity(
	req *network.RevokeExternalConnectivityRequest) error {
	log.Debugln("=== RevokeExternalConnectivity")
	log.Debugln(req)
	return nil
}

func (d *ContrailDriver) createRootNetwork() error {
	// HNS automatically creates a new vswitch if the first HNS network is created. We want to
	// control this behaviour. That's why we create a dummy root HNS network.

	rootNetwork, err := hns.GetHNSNetworkByName(common.RootNetworkName)
	if err != nil {
		return err
	}
	if rootNetwork == nil {

		subnets := []hcsshim.Subnet{
			{
				AddressPrefix: "0.0.0.0/24",
			},
		}
		configuration := &hcsshim.HNSNetwork{
			Name:               common.RootNetworkName,
			Type:               "transparent",
			NetworkAdapterName: string(d.networkAdapter),
			Subnets:            subnets,
		}
		rootNetID, err := hns.CreateHNSNetwork(configuration)
		if err != nil {
			return err
		}

		log.Infoln("Created root HNS network:", rootNetID)
	} else {
		log.Infoln("Existing root HNS network found:", rootNetwork.Id)
	}
	return nil
}

func (d *ContrailDriver) waitForPipeToStart() error {
	return d.waitForPipe(true)
}

func (d *ContrailDriver) waitForPipeToStop() error {
	return d.waitForPipe(false)
}

func (d *ContrailDriver) waitForPipe(waitUntilExists bool) error {
	timeStarted := time.Now()
	for {
		if time.Since(timeStarted) > time.Millisecond*common.PipePollingTimeout {
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

		time.Sleep(time.Millisecond * common.PipePollingRate)
	}

	time.Sleep(time.Second * 1)

	if waitUntilExists {
		return d.waitUntilPipeDialable()
	}

	return nil
}

func (d *ContrailDriver) waitUntilPipeDialable() error {
	timeStarted := time.Now()
	for {
		if time.Since(timeStarted) > time.Millisecond*common.PipePollingTimeout {
			return errors.New("Waited for pipe to be dialable for too long.")
		}

		timeout := time.Millisecond * 10
		conn, err := sockets.DialPipe(d.PipeAddr, timeout)
		if err == nil {
			conn.Close()
			return nil
		}

		log.Errorf("Waiting until dialable, but: %s", err)

		time.Sleep(time.Millisecond * common.PipePollingRate)
	}
}

func (d *ContrailDriver) networkMetaFromDockerNetwork(dockerNetID string) (*NetworkMeta,
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

func (d *ContrailDriver) dockerNetworksMeta() ([]NetworkMeta, error) {
	var meta []NetworkMeta

	docker, err := dockerClient.NewEnvClient()
	if err != nil {
		return nil, err
	}

	netList, err := docker.NetworkList(context.Background(), dockerTypes.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	for _, net := range netList {
		tenantContrail, tenantExists := net.Options["tenant"]
		networkContrail, networkExists := net.Options["network"]
		if tenantExists && networkExists {
			meta = append(meta, NetworkMeta{
				tenant:     tenantContrail,
				network:    networkContrail,
				subnetCIDR: net.IPAM.Config[0].Subnet,
			})
		}
	}
	return meta, nil
}

func (d *ContrailDriver) hnsNetworksMeta() ([]NetworkMeta, error) {
	hnsNetworks, err := d.hnsMgr.ListNetworks()
	if err != nil {
		return nil, err
	}

	var meta []NetworkMeta
	for _, net := range hnsNetworks {
		splitName := strings.Split(net.Name, ":")
		// hnsManager.ListNetworks() already sanitizes network name
		tenantName := splitName[1]
		networkName := splitName[2]
		subnetCIDR := splitName[3]
		meta = append(meta, NetworkMeta{
			tenant:     tenantName,
			network:    networkName,
			subnetCIDR: subnetCIDR,
		})
	}
	return meta, nil
}

func (d *ContrailDriver) generateFriendlyName(hnsEndpointID string) string {
	// Here's how the Forwarding Extension (kernel) can identify interfaces based on their
	// friendly names.
	// Windows Containers have NIC names like "NIC ID abcdef", where abcdef are the first 6 chars
	// of their HNS endpoint ID.
	// Hyper-V Containers have NIC names consisting of two uuids, probably representing utitlity
	// VM's interface and endpoint's interface:
	// "227301f6-bee9-4ae2-8a93-5e900cde3f47--910c5490-bff8-45e3-a2a0-0114ed9903e0"
	// The second UUID (after the "--") is the HNS endpoints ID.

	// For now, we will always send the name in the Windows Containers format, because it probably
	// has enough information to recognize it in kernel (6 first chars of UUID should be enough):
	containerNicID := strings.Split(hnsEndpointID, "-")[0]
	return fmt.Sprintf("Container NIC %s", containerNicID)
}

func (d *ContrailDriver) getContrailSubnetCIDR(ipam *types.IpamSubnetType) string {
	return fmt.Sprintf("%s/%v", ipam.Subnet.IpPrefix, ipam.Subnet.IpPrefixLen)
}
