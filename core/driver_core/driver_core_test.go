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

package driver_core_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	netSim "github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/simulator"
	"github.com/Juniper/contrail-windows-docker-driver/agent"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

const (
	tenantName  = "agatka"
	networkName = "test_net"
	subnetCIDR  = "1.2.3.4/24"
)

func TestCore(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("core_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Core test suite",
		[]Reporter{junitReporter})
}

var _ = Describe("Core tests", func() {
	var testedCore *driver_core.ContrailDriverCore
	var controller ports.Controller
	var localNetRepo ports.LocalContrailNetworkRepository

	BeforeEach(func() {
		testedCore, controller, localNetRepo = newSimulatedModulesUnderTest()
	})

	Context("CreateNetwork", func() {
		BeforeEach(func() {
			_ = testProject(controller)
			_ = testNetwork(controller)
		})
		It("responds with nil", func() {
			err := testedCore.CreateNetwork(tenantName, networkName, subnetCIDR)
			Expect(err).ToNot(HaveOccurred())
		})
		It("creates a local Contrail network", func() {
			netsBefore, err := localNetRepo.ListNetworks()
			Expect(err).ToNot(HaveOccurred())

			err = testedCore.CreateNetwork(tenantName, networkName, subnetCIDR)

			Expect(err).ToNot(HaveOccurred())
			netsAfter, err := localNetRepo.ListNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(netsBefore).To(HaveLen(len(netsAfter) - 1))
		})

		type TestCase struct {
			tenant  string
			network string
		}
		DescribeTable("using resources that don't exist in Controller",
			func(t TestCase) {
				err := testedCore.CreateNetwork(t.tenant, t.network, subnetCIDR)
				Expect(err).To(HaveOccurred())
			},
			Entry("no such subnet resource", TestCase{
				tenant:  tenantName,
				network: "nonexistingNetwork",
			}),
			Entry("no such tenant resource", TestCase{
				tenant:  "nonexistingTenant",
				network: networkName,
			}),
		)
	})

	Context("DeleteNetwork", func() {
	})

	Context("CreateEndpoint", func() {
	})

	Context("DeleteEndpoint", func() {
	})
})

func newSimulatedModulesUnderTest() (c *driver_core.ContrailDriverCore, controller ports.Controller,
	netRepo ports.LocalContrailNetworkRepository) {
	ext := &hyperv_extension.HyperVExtensionSimulator{
		Enabled: false,
		Running: true,
	}
	vrouter := vrouter.NewHyperVvRouter(ext)

	controller = controller_rest.NewFakeControllerAdapter()

	netRepo = netSim.NewInMemContrailNetworksRepository()
	epRepo := &netSim.InMemEndpointRepository{}

	// TODO: Implement simulator for Agent.
	serverUrl, _ := url.Parse("http://127.0.0.1:9091")
	agent := agent.NewAgentRestAPI(http.DefaultClient, serverUrl)

	var err error
	c, err = driver_core.NewContrailDriverCore(vrouter, controller, agent, netRepo, epRepo)
	Expect(err).ToNot(HaveOccurred())

	return
}

func testProject(c ports.Controller) *types.Project {
	project, err := c.NewProject(common.DomainName, tenantName)
	Expect(err).ToNot(HaveOccurred())
	return project
}

func testNetwork(c ports.Controller) *types.VirtualNetwork {
	network, err := c.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)
	Expect(err).ToNot(HaveOccurred())
	return network
}
