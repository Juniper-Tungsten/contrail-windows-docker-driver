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

package driver_core

import (
	"errors"
	"fmt"

	// TODO: this import should be removed when making Controller port smaller
	"github.com/Juniper/contrail-go-api/types"

	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	log "github.com/sirupsen/logrus"
)

type ContrailDriverCore struct {
	vrouter ports.VRouter
	// TODO: all these fields below should be made private as we remove the need for them in
	// driver package.
	Controller                 ports.Controller
	Agent                      ports.Agent
	LocalContrailNetworksRepo  ports.LocalContrailNetworkRepository
	LocalContrailEndpointsRepo ports.LocalContrailEndpointRepository
}

func NewContrailDriverCore(vr ports.VRouter, c ports.Controller, a ports.Agent,
	nr ports.LocalContrailNetworkRepository,
	er ports.LocalContrailEndpointRepository) (*ContrailDriverCore, error) {
	core := ContrailDriverCore{
		vrouter:    vr,
		Controller: c,
		Agent:      a,
		LocalContrailNetworksRepo:  nr,
		LocalContrailEndpointsRepo: er,
	}
	if err := core.initializeAdapters(); err != nil {
		return nil, err
	}
	return &core, nil
}

func (core *ContrailDriverCore) initializeAdapters() error {
	return core.vrouter.Initialize()
}

func (core *ContrailDriverCore) CreateNetwork(tenantName, networkName, ipPool string) error {
	// Check if network is already created in Contrail.
	log.Infoln(tenantName, networkName)
	contrailNetwork, err := core.Controller.GetNetwork(tenantName, networkName)
	if err != nil {
		return err
	}
	if contrailNetwork == nil {
		return errors.New("Retrieved Contrail network is nil")
	}

	log.Infoln("Got Contrail network", contrailNetwork.GetDisplayName())

	contrailIpam, err := core.Controller.GetIpamSubnet(contrailNetwork, ipPool)
	if err != nil {
		return err
	}
	subnetCIDR := core.GetContrailSubnetCIDR(contrailIpam)

	contrailGateway := contrailIpam.DefaultGateway
	if contrailGateway == "" {
		// TODO: this fails in unit tests using contrail-go-api mock. So either:
		// * fix contrail-go-api mock to return *some* default GW
		// * or maybe we should keep going if default GW is empty, as a user may wish not
		//   to specify it. (this is what we do now, and just generate a warning)
		log.Warn("Default GW is empty")
	}

	// TODO: all the statements above should probably be refactored into a single method of
	// Controller port. Something like
	// subnetCIDR, gateway := GetSubnet(tenantName, networkName, ipPool)

	_, err = core.LocalContrailNetworksRepo.CreateNetwork(tenantName, networkName, subnetCIDR,
		contrailGateway)

	return err
}

func (core *ContrailDriverCore) DeleteNetwork(tenantName, networkName, subnetCIDR string) error {
	return core.LocalContrailNetworksRepo.DeleteNetwork(tenantName, networkName, subnetCIDR)
}

func (core *ContrailDriverCore) GetContrailSubnetCIDR(ipam *types.IpamSubnetType) string {
	// TODO: this should be probably moved to Controller when we make Controller port smaller.
	return fmt.Sprintf("%s/%v", ipam.Subnet.IpPrefix, ipam.Subnet.IpPrefixLen)
}
