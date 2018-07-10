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

package hns

import (
	"strings"

	"github.com/Juniper/contrail-windows-docker-driver/core/model"
	"github.com/Microsoft/hcsshim"
)

type HNSEndpointRepository struct{}

func (repo *HNSEndpointRepository) CreateEndpoint(name string, container *model.Container, network *model.Network) (string, error) {
	// Ensure that MAC address passed to HNS if foramtted in correct way.
	// Contrail MACs are like 11:22:aa:bb:cc:dd
	// HNS needs MACs like 11-22-AA-BB-CC-DD
	containerMac := strings.Replace(strings.ToUpper(container.Mac), ":", "-", -1)

	configuration := &hcsshim.HNSEndpoint{
		Name:           name,
		VirtualNetwork: network.LocalID,
		IPAddress:      container.IP,
		GatewayAddress: container.Gateway,
		MacAddress:     containerMac,
	}
	return CreateHNSEndpoint(configuration)
}

func (repo *HNSEndpointRepository) GetEndpointByName(name string) (*hcsshim.HNSEndpoint, error) {
	return GetHNSEndpointByName(name)
}

func (repo *HNSEndpointRepository) DeleteEndpoint(endpointID string) error {
	return DeleteHNSEndpoint(endpointID)
}
