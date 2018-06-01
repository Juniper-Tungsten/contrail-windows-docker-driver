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

package hnsManager_integration_test

import (
	"flag"
	"fmt"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hns"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hns/win_networking"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/hnsManager"
	"github.com/Juniper/contrail-windows-docker-driver/integration_tests/helpers"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var netAdapter string

func init() {
	flag.StringVar(&netAdapter, "netAdapter", "Ethernet0", "Ethernet adapter name to use")
	log.SetLevel(log.DebugLevel)
}

func TestHNSManager(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("hnsManager_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "HNS manager test suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	// Code disabled: cannot mark 'BeforeSuite' block as Pending...
	// err := common.HardResetHNS()
	// Expect(err).ToNot(HaveOccurred())
	// err = win_networking.WaitForValidIPReacquisition(common.AdapterName(netAdapter))
	// Expect(err).ToNot(HaveOccurred())
})

var _ = PDescribe("HNS manager", func() {

	const (
		tenantName  = "agatka"
		networkName = "test_net"
		subnetCIDR  = "10.0.0.0/24"
		defaultGW   = "10.0.0.1"
	)

	var hnsMgr *hnsManager.HNSManager

	BeforeEach(func() {
		hnsMgr = &hnsManager.HNSManager{}
	})

	AfterEach(func() {
		err := common.HardResetHNS()
		Expect(err).ToNot(HaveOccurred())
		err = win_networking.WaitForValidIPReacquisition(common.AdapterName(netAdapter))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("specified network does not exist", func() {
		Specify("creating a new HNS network works", func() {
			_, err := hnsMgr.CreateNetwork(common.AdapterName(netAdapter), tenantName, networkName,
				subnetCIDR, defaultGW)
			Expect(err).ToNot(HaveOccurred())
		})
		Specify("getting the HNS network returns error", func() {
			net, err := hnsMgr.GetNetwork(tenantName, networkName, subnetCIDR)
			Expect(err).To(HaveOccurred())
			Expect(net).To(BeNil())
		})
	})

	Context("specified network already exists", func() {
		var existingNetID string
		BeforeEach(func() {
			hnsNetName := fmt.Sprintf("Contrail:%s:%s:%s", tenantName, networkName, subnetCIDR)
			existingNetID = helpers.CreateTestHNSNetwork(common.AdapterName(netAdapter), hnsNetName,
				subnetCIDR, defaultGW)
		})

		Specify("creating a new network with same params returns error", func() {
			net, err := hnsMgr.CreateNetwork(common.AdapterName(netAdapter), tenantName,
				networkName, subnetCIDR, defaultGW)
			Expect(err).To(HaveOccurred())
			Expect(net).To(BeNil())
		})

		Specify("getting the network returns it", func() {
			net, err := hnsMgr.GetNetwork(tenantName, networkName, subnetCIDR)
			Expect(err).ToNot(HaveOccurred())
			Expect(net.Id).To(Equal(existingNetID))
		})

		Context("network has active endpoints", func() {
			BeforeEach(func() {
				eps, err := hns.ListHNSEndpoints()
				Expect(err).ToNot(HaveOccurred())
				Expect(eps).To(BeEmpty())

				_ = helpers.CreateTestHNSEndpoint(existingNetID)

				eps, err = hns.ListHNSEndpoints()
				Expect(err).ToNot(HaveOccurred())
				Expect(eps).ToNot(BeEmpty())
			})

			Specify("deleting the network returns error", func() {
				err := hnsMgr.DeleteNetwork(tenantName, networkName, subnetCIDR)
				Expect(err).To(HaveOccurred())

				eps, err := hns.ListHNSEndpoints()
				Expect(err).ToNot(HaveOccurred())
				Expect(eps).ToNot(BeEmpty())
			})
		})

		Context("network has no active endpoints", func() {
			Specify("deleting the network removes it", func() {
				netsBefore, err := hns.ListHNSNetworks()
				Expect(err).ToNot(HaveOccurred())
				err = hnsMgr.DeleteNetwork(tenantName, networkName, subnetCIDR)
				Expect(err).ToNot(HaveOccurred())
				netsAfter, err := hns.ListHNSNetworks()
				Expect(err).ToNot(HaveOccurred())
				Expect(netsBefore).To(HaveLen(len(netsAfter) + 1))
			})
		})
	})

	Describe("Listing Contrail networks", func() {
		BeforeEach(func() {
			names := []string{
				fmt.Sprintf("Contrail:%s:%s:%s", "tenant1", "netname1", "1.2.3.4/24"),
				fmt.Sprintf("Contrail:%s:%s:%s", "tenant2", "netname2", "2.3.4.5/24"),
				fmt.Sprintf("Contrail:%s", "invalid_num_of_fields"),
				fmt.Sprintf("Contrail:%s:%s", "invalid", "num_of_fields"),
				fmt.Sprintf("Contrail:%s:%s:%s:%s", "invalid", "num", "of", "fields"),
				"some_other_name",
			}
			for _, n := range names {
				helpers.CreateTestHNSNetwork(common.AdapterName(netAdapter), n, subnetCIDR, defaultGW)
			}
		})
		Specify("Listing only Contrail networks works", func() {
			nets, err := hnsMgr.ListNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).To(HaveLen(2))
			for _, n := range nets {
				Expect(n.Name).To(ContainSubstring("Contrail:"))
			}
		})
	})
})
