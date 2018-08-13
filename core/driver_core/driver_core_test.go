// +build unit
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
	"net"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/core/model"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	netSim "github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/simulator"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/port_association/agent"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

const (
	tenantName        = "agatka"
	networkName       = "test_net"
	securityGroupName = "default"
	subnetCIDR        = "1.2.3.0/24"
	dockerNetID       = "1234dnID"
	endpointID        = "5678epID"
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
	var localEpRepo ports.LocalContrailEndpointRepository

	BeforeEach(func() {
		testedCore, controller, localNetRepo, localEpRepo = newSimulatedModulesUnderTest()
	})

	Context("CreateNetwork", func() {
		BeforeEach(func() {
			_ = testProject(controller)
			_ = testNetwork(controller)
		})
		It("responds with nil", func() {
			err := testedCore.CreateNetwork(dockerNetID, tenantName, networkName, subnetCIDR)
			Expect(err).ToNot(HaveOccurred())
		})
		It("creates a local Contrail network", func() {
			netsBefore, err := localNetRepo.ListNetworks()
			Expect(err).ToNot(HaveOccurred())

			err = testedCore.CreateNetwork(dockerNetID, tenantName, networkName, subnetCIDR)

			Expect(err).ToNot(HaveOccurred())
			netsAfter, err := localNetRepo.ListNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(netsBefore).To(HaveLen(len(netsAfter) - 1))
		})
		It("gets CIDR information from Contrail if it's unspecified in given network", func() {
			unspecifiedCIDR := "0.0.0.0/0"
			err := testedCore.CreateNetwork(dockerNetID, tenantName, networkName, unspecifiedCIDR)
			Expect(err).ToNot(HaveOccurred())
			net, err := localNetRepo.GetNetwork(dockerNetID)
			Expect(err).ToNot(HaveOccurred())
			Expect(net.Subnet.CIDR).To(Equal(subnetCIDR))
		})
		type TestCase struct {
			tenant  string
			network string
		}
		DescribeTable("using resources that don't exist in Controller",
			func(t TestCase) {
				err := testedCore.CreateNetwork(dockerNetID, t.tenant, t.network, subnetCIDR)
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

	setupControllerNetworkAndLocalNetwork := func() {
		_ = testProject(controller)
		_ = testNetwork(controller)
		err := testedCore.CreateNetwork(dockerNetID, tenantName, networkName, subnetCIDR)
		Expect(err).ToNot(HaveOccurred())
	}
	setupControllerNetworkWithoutLocalNetwork := func() {
		_ = testProject(controller)
		_ = testNetwork(controller)
	}
	setupLocalNetworkWithoutControllerNetwork := func() {
		someGateway := "1.2.3.1"
		subnet := model.Subnet{
			CIDR:      subnetCIDR,
			DefaultGW: someGateway,
		}
		net := model.Network{
			TenantName:  tenantName,
			NetworkName: networkName,
			Subnet:      subnet,
		}
		err := localNetRepo.CreateNetwork(dockerNetID, &net)
		Expect(err).ToNot(HaveOccurred())
	}

	Context("DeleteNetwork", func() {

		assertReturnsError := func() {
			err := testedCore.DeleteNetwork(dockerNetID)
			Expect(err).To(HaveOccurred())
		}
		assertDoesNotError := func() {
			err := testedCore.DeleteNetwork(dockerNetID)
			Expect(err).ToNot(HaveOccurred())
		}
		assertRemovesLocalNetwork := func() {
			_ = testedCore.DeleteNetwork(dockerNetID)

			netsAfter, err := localNetRepo.ListNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(netsAfter).To(HaveLen(0))
		}
		assertDoesNotRemoveControllerNetwork := func() {
			_ = testedCore.DeleteNetwork(dockerNetID)
			net, err := controller.GetNetworkWithSubnet(tenantName, networkName, subnetCIDR)

			Expect(err).ToNot(HaveOccurred())
			Expect(net).ToNot(BeNil())
		}
		Context("Controller network and local network exist", func() {
			BeforeEach(setupControllerNetworkAndLocalNetwork)
			It("does not error", assertDoesNotError)
			It("removes local network", assertRemovesLocalNetwork)
			It("doesn't remove Controller network", assertDoesNotRemoveControllerNetwork)
		})
		Context("Controller network exists, but local network does not exist", func() {
			BeforeEach(setupControllerNetworkWithoutLocalNetwork)
			It("returns an error", assertReturnsError)
			It("doesn't remove Controller network", assertDoesNotRemoveControllerNetwork)
		})
		Context("Controller network does not exist, but local network does", func() {
			BeforeEach(setupLocalNetworkWithoutControllerNetwork)
			It("does not error", assertDoesNotError)
			It("removes local network", assertRemovesLocalNetwork)
		})
		PContext("network has active endpoints", func() {
			// TODO: marked as pending, because simulator doesn't check for active endpoints yet.
			// "actual" HNS implementation uses global HNS state to retreive the list of
			// endpoints. To do such thing in simulator, we would have to pass
			// InMemEndpointRepository repository to InMemContrailNetworksRepository or
			// refactor them in some way. Let's defer such refactor to when refactoring
			// DeleteEndpoint request.
			BeforeEach(func() {
				setupControllerNetworkAndLocalNetwork()
				_, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
				Expect(err).ToNot(HaveOccurred())
			})
			It("returns an error", assertReturnsError)
			It("does not remove local network", func() {
				err := testedCore.DeleteNetwork(dockerNetID)
				Expect(err).To(HaveOccurred())

				netsAfter, err := localNetRepo.ListNetworks()
				Expect(err).ToNot(HaveOccurred())
				Expect(netsAfter).To(HaveLen(1))
			})
		})
	})

	Context("CreateEndpoint", func() {
		var recvChan chan interface{}
		var server *ghttp.Server
		BeforeEach(func() {
			recvChan = make(chan interface{})
			server = testServer(recvChan)
			server.AppendHandlers(
				ghttp.VerifyRequest("POST", "/port"),
				ghttp.RespondWith(http.StatusOK, ""),
			)
		})
		AfterEach(func() {
			server.Close()
		})

		Context("Controller network and local network exist", func() {
			BeforeEach(setupControllerNetworkAndLocalNetwork)
			AfterEach(func() {
				// Because right now port request is send asynchronously in a goroutine, we need to
				// wait after each test case for any requests before moving onto the next test case.
				// This is to ensure test isolation. Otherwise, the async request may "spill over" to
				// the next test case which would be hard to debug.
				By("notifies port listener about association")
				Eventually(func() []*http.Request {
					return server.ReceivedRequests()
				}).Should(HaveLen(1))
			})
			It("returns container resource allocated in controller", func() {
				container, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
				Expect(err).ToNot(HaveOccurred())

				Expect(container.IP).To(MatchRegexp(`1.2.3.[0-9]+`))
				Expect(container.PrefixLen).To(Equal(24))
				Expect(container.Mac).To(MatchRegexp(`([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})`))
				Expect(container.VmUUID).ToNot(Equal(""))
				Expect(container.VmiUUID).ToNot(Equal(""))
			})
			It("configures HNS endpoint", func() {
				_, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
				Expect(err).ToNot(HaveOccurred())

				ep, err := localEpRepo.GetEndpoint(endpointID)
				Expect(err).ToNot(HaveOccurred())
				Expect(ep).ToNot(BeNil())
				Expect(ep.Name).To(Equal(endpointID))
			})
		})

		assertReturnsError := func() {
			container, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
			Expect(err).To(HaveOccurred())
			Expect(container).To(BeNil())
		}
		assertDoesNotAllocate := func() {
			container, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
			Expect(err).To(HaveOccurred())
			Expect(container).To(BeNil())
		}
		assertDoesNotConfigure := func() {
			_, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
			Expect(err).To(HaveOccurred())

			ep, err := localEpRepo.GetEndpoint(endpointID)
			Expect(err).To(HaveOccurred())
			Expect(ep).To(BeNil())
		}
		assertAfterEachDoesNotNotifyAboutAssociation := func() {
			// Because right now port request is send asynchronously in a goroutine, we need to
			// wait after each test case for any requests before moving onto the next test case.
			// This is to ensure test isolation. Otherwise, the async request may "spill over" to
			// the next test case which would be hard to debug.
			By("does not notify port listener about association")
			Consistently(func() []*http.Request {
				return server.ReceivedRequests()
			}).Should(HaveLen(0))
		}
		Context("Controller network exists, but local network does not exist", func() {
			BeforeEach(setupControllerNetworkWithoutLocalNetwork)
			AfterEach(assertAfterEachDoesNotNotifyAboutAssociation)
			It("returns an error", assertReturnsError)
			It("doesn't allocate resources in controller", assertDoesNotAllocate)
			It("doesn't configure HNS endpoint", assertDoesNotConfigure)
		})
		Context("Controller network does not exist, but local network does", func() {
			BeforeEach(setupLocalNetworkWithoutControllerNetwork)
			AfterEach(assertAfterEachDoesNotNotifyAboutAssociation)
			It("returns an error", assertReturnsError)
			It("doesn't allocate resources in controller", assertDoesNotAllocate)
			It("doesn't configure HNS endpoint", assertDoesNotConfigure)
		})
	})

	Context("DeleteEndpoint", func() {
		var recvChan chan interface{}
		var server *ghttp.Server
		BeforeEach(func() {
			recvChan = make(chan interface{})
			server = testServer(recvChan)
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/port"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			// We need to use RouteToHandler method here, because it accepts regex paths.
			server.RouteToHandler("DELETE", regexp.MustCompile(`port/.*`), ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusOK, ""),
			))
		})
		AfterEach(func() {
			server.Close()
		})

		setupLocalEndpointAndContainerInController := func() {
			setupControllerNetworkAndLocalNetwork()
			_, err := testedCore.CreateEndpoint(dockerNetID, endpointID)
			Expect(err).ToNot(HaveOccurred())

			// wait for port association request to arrive before continuing, otherwise there is
			// a possible race condition with port disassociation request.
			Eventually(func() []*http.Request {
				return server.ReceivedRequests()
			}).Should(HaveLen(1))
		}

		assertReturnsError := func() {
			err := testedCore.DeleteEndpoint(dockerNetID, endpointID)
			Expect(err).To(HaveOccurred())
		}
		assertRemovesResource := func() {
			_ = testedCore.DeleteEndpoint(dockerNetID, endpointID)

			vm, err := controller.GetInstance(endpointID)
			Expect(err).To(HaveOccurred())
			Expect(vm).To(BeNil())
		}
		assertAfterEachNotifiesAboutDissociation := func() {
			By("notifies port listener about dissacociation")
			// we expect two requests to have arrived: first for port association in test setup;
			// the other is the one we actually look for.
			Eventually(func() []*http.Request {
				return server.ReceivedRequests()
			}).Should(HaveLen(2))
		}

		Context("Local endpoint and Controller resource exist", func() {
			BeforeEach(setupLocalEndpointAndContainerInController)
			AfterEach(assertAfterEachNotifiesAboutDissociation)
			It("does not error", func() {
				err := testedCore.DeleteEndpoint(dockerNetID, endpointID)
				Expect(err).ToNot(HaveOccurred())
			})
			It("removes local endpoint", func() {
				_ = testedCore.DeleteEndpoint(dockerNetID, endpointID)

				ep, err := localEpRepo.GetEndpoint(endpointID)
				Expect(ep).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
			It("removes resource from Controller", assertRemovesResource)
		})
		Context("Only resource in Controller exists", func() {
			BeforeEach(func() {
				setupLocalEndpointAndContainerInController()
				err := localEpRepo.DeleteEndpoint(endpointID)
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(assertAfterEachNotifiesAboutDissociation)
			It("returns an error", assertReturnsError)
			It("removes resource from Controller", assertRemovesResource)
		})
		Context("Only local endpoint exists", func() {
			BeforeEach(func() {
				setupLocalEndpointAndContainerInController()
				err := controller.DeleteContainer(endpointID)
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				By("does not notify port listener about dissacociation")
				// we expect only one request to have arrived - for port association in setup;
				Consistently(func() []*http.Request {
					return server.ReceivedRequests()
				}).Should(HaveLen(1))
			})
			It("returns an error", assertReturnsError)
			It("does not remove local endpoint", func() {
				_ = testedCore.DeleteEndpoint(dockerNetID, endpointID)

				ep, err := localEpRepo.GetEndpoint(endpointID)
				Expect(ep).ToNot(BeNil())
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
})

func newSimulatedModulesUnderTest() (c *driver_core.ContrailDriverCore, controller ports.Controller,
	netRepo ports.LocalContrailNetworkRepository,
	epRepo ports.LocalContrailEndpointRepository) {
	ext := &hyperv_extension.HyperVExtensionSimulator{
		Enabled: false,
		Running: true,
	}
	vrouter := vrouter.NewHyperVvRouter(ext)

	controller = controller_rest.NewFakeControllerAdapter()

	netRepo = netSim.NewInMemContrailNetworksRepository()
	epRepo = netSim.NewInMemEndpointRepository()

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

func testServer(recv chan interface{}) *ghttp.Server {
	// TODO: Refactor this test to use listener simulator, instead this test
	// http server, when it's implemented.
	server := ghttp.NewUnstartedServer()
	listener, err := net.Listen("tcp", "127.0.0.1:9091")
	Expect(err).ToNot(HaveOccurred())
	server.HTTPTestServer.Listener = listener

	server.Start()
	return server
}
