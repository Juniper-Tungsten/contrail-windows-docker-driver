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

package vmswitch

import (
	"fmt"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hns"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/win_networking"
	"github.com/Juniper/contrail-windows-docker-driver/powershell"
	"github.com/Microsoft/hcsshim"
	log "github.com/sirupsen/logrus"
)

const (
	// rootNetworkName is a name of root HNS network created solely for the purpose of
	// having a virtual switch
	rootNetworkName = "ContrailRootNetwork"
)

func DoesSwitchExist(name string) (bool, error) {
	c := []string{"Get-VMSwitch", "-Name", fmt.Sprintf("\"%s\"", name)}
	stdout, _, err := powershell.CallPowershell(c...)
	if err != nil {
		return false, err
	} else if stdout == "" {
		return false, nil
	} else {
		return true, nil
	}
}

func EnsureSwitchExists(vmSwitchName, nameOfAdapterToUse string) error {
	// HNS automatically creates a new vswitch if the first HNS network is created. We want to
	// control this behaviour. That's why we create a dummy root HNS network.

	rootNetwork, err := hns.GetHNSNetworkByName(rootNetworkName)
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
			Name:               rootNetworkName,
			Type:               "transparent",
			NetworkAdapterName: nameOfAdapterToUse,
			Subnets:            subnets,
		}
		// Before we CreateHNSNetwork we need to make sure, that interface we want to attach the vmswitch
		// to has correct IP address. Otherwise, HNS will complain. The interface exists only, if root HNS
		// network doesn't yet exist. It disappears the moment vmswitch is created.
		ext, _ := DoesSwitchExist(vmSwitchName)
		if !ext {
			if err := win_networking.WaitForValidIPReacquisition(nameOfAdapterToUse); err != nil {
				return err
			}
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
