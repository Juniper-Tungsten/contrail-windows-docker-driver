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

	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libnetwork/netlabel"
	log "github.com/sirupsen/logrus"
)

func (d *ServerCNM) GetCapabilities() (*network.CapabilitiesResponse, error) {
	log.Debugln("Received GetCapabilities request from docker daemon")
	r := &network.CapabilitiesResponse{}
	r.Scope = network.LocalScope
	return r, nil
}

func (d *ServerCNM) CreateNetwork(req *network.CreateNetworkRequest) error {
	log.Debugln("Received CreateNetwork request from docker daemon:", req)

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
	log.Debugln("Received AllocateNetwork request from docker daemon:", req)
	// This method is used in swarm, in remote plugins. We don't implement it.
	return nil, errors.New("AllocateNetwork is not implemented")
}

func (d *ServerCNM) DeleteNetwork(req *network.DeleteNetworkRequest) error {
	log.Debugln("Received DeleteNetwork request from docker daemon:", req)

	return d.Core.DeleteNetwork(req.NetworkID)
}

func (d *ServerCNM) FreeNetwork(req *network.FreeNetworkRequest) error {
	log.Debugln("Received FreeNetwork request from docker daemon:", req)

	// This method is used in swarm, in remote plugins. We don't implement it.
	return errors.New("FreeNetwork is not implemented")
}

func (d *ServerCNM) CreateEndpoint(req *network.CreateEndpointRequest) (
	*network.CreateEndpointResponse, error) {
	log.Debugln("Received CreateEndpoint request from docker daemon:", req)

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
	log.Debugln("Received DeleteEndpoint request from docker daemon:", req)

	return d.Core.DeleteEndpoint(req.NetworkID, req.EndpointID)
}

func (d *ServerCNM) EndpointInfo(req *network.InfoRequest) (*network.InfoResponse, error) {
	log.Debugln("Received EndpointInfo request from docker daemon:", req)

	hnsEpName := req.EndpointID
	hnsEp, err := d.Core.LocalContrailEndpointsRepo.GetEndpoint(hnsEpName)
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
	log.Debugln("Received Join request from docker daemon:", req)

	hnsEp, err := d.Core.LocalContrailEndpointsRepo.GetEndpoint(req.EndpointID)
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
	log.Debugln("Received Leave request from docker daemon:", req)

	hnsEp, err := d.Core.LocalContrailEndpointsRepo.GetEndpoint(req.EndpointID)
	if err != nil {
		return err
	}
	if hnsEp == nil {
		return errors.New("Such HNS endpoint doesn't exist")
	}

	return nil
}

func (d *ServerCNM) DiscoverNew(req *network.DiscoveryNotification) error {
	log.Debugln("Received DiscoverNew request from docker daemon:", req)
	// We don't care about discovery notifications.
	return nil
}

func (d *ServerCNM) DiscoverDelete(req *network.DiscoveryNotification) error {
	log.Debugln("Received DiscoverDelete request from docker daemon:", req)
	// We don't care about discovery notifications.
	return nil
}

func (d *ServerCNM) ProgramExternalConnectivity(
	req *network.ProgramExternalConnectivityRequest) error {
	log.Debugln("Received ProgramExternalConnectivity request from docker daemon:", req)
	return nil
}

func (d *ServerCNM) RevokeExternalConnectivity(
	req *network.RevokeExternalConnectivityRequest) error {
	log.Debugln("Received RevokeExternalConnectivity request from docker daemon:", req)
	return nil
}
