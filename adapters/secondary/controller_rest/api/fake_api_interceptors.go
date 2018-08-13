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

package api

import (
	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/types"
	log "github.com/sirupsen/logrus"
)

// vnInterceptor is responsible for adding a DefaultGateway to ipamSubnets created in
// tests. This is mimicking the behaviour of the actual API server that assings this
// address on its own.
type vnInterceptor struct{}

func (ceptor *vnInterceptor) Get(obj contrail.IObject) {
}

func (ceptor *vnInterceptor) Put(obj contrail.IObject) {
	vn, ok := obj.(*types.VirtualNetwork)
	if !ok {
		log.Errorln("Invalid cast")
	}
	ipamReferences, err := vn.GetNetworkIpamRefs()
	if err != nil {
		log.Errorln(err)
		return
	}
	for refIdx, _ := range ipamReferences {
		ipamSubnets := ipamReferences[refIdx].Attr.(types.VnSubnetsType).IpamSubnets
		for ipamIdx, _ := range ipamSubnets {
			// It doesn't matter what IP we put here, because we don't care in the
			// tests. What matters, is that it's not an empty string - for now.
			ipamSubnets[ipamIdx].DefaultGateway = "1.2.3.4"
		}
	}
}

// vmiInterceptor is responsible for assigning a MAC address to retreived virtual machine
// interface. It mimicks the behaviour of the actual API server.
type vmiInterceptor struct{}

func (ceptor *vmiInterceptor) Get(obj contrail.IObject) {
	vmi, ok := obj.(*types.VirtualMachineInterface)
	if !ok {
		log.Errorln("Invalid cast")
	}
	macs := vmi.GetVirtualMachineInterfaceMacAddresses()
	if len(macs.MacAddress) == 0 {
		// Add new MAC address - it doesn't matter what the actual value is, as we don't
		// check it in the tests - for now.
		var newMacs types.MacAddressesType
		newMacs.AddMacAddress("DE:AD:BE:EF:FE:ED")
		vmi.SetVirtualMachineInterfaceMacAddresses(&newMacs)
	}
}

func (ceptor *vmiInterceptor) Put(obj contrail.IObject) {
}

// ipInterceptor is responsible for assigning an IP to newly created interface. This mimics
// the behaviour of the actual APi server.
type iipInterceptor struct{}

func (ceptor *iipInterceptor) Get(obj contrail.IObject) {
	ipobj, ok := obj.(*types.InstanceIp)
	if !ok {
		log.Errorln("Invalid cast")
	}
	ips := ipobj.GetInstanceIpAddress()
	if len(ips) == 0 {
		// We don't care about the specific value of IP, because we don't compare it in tests
		// - for now.
		ipobj.SetInstanceIpAddress("1.2.3.4")
	}

}

func (ceptor *iipInterceptor) Put(obj contrail.IObject) {
}
