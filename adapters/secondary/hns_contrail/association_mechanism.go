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

package hns_contrail

import (
	"fmt"
	"strings"
)

const (
	// hnsNetworkPrefix is a prefix given too all HNS network names managed by the driver
	hnsNetworkPrefix = "Contrail"
)

// HNSDBNetworkAssociationMechanism is used for associating docker, HNS and Contrail networks
// in HNS database.
type HNSDBNetworkAssociationMechanism struct{}

func (m HNSDBNetworkAssociationMechanism) GenerateName(dockerNetID, contrailTenantName, contrailNetworkName, contrailSubnetCIDR string) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", hnsNetworkPrefix, dockerNetID, contrailTenantName,
		contrailNetworkName, contrailSubnetCIDR)
}

func (m HNSDBNetworkAssociationMechanism) SplitName(name string) (dockerNetID, tenantName, networkName, subnetCIDR string) {
	split := strings.Split(name, ":")
	dockerNetID = split[1]
	tenantName = split[2]
	networkName = split[3]
	subnetCIDR = split[4]
	return
}

func (m HNSDBNetworkAssociationMechanism) IsOwnedByDriver(name string) bool {
	split := strings.Split(name, ":")
	return split[0] == hnsNetworkPrefix
}
