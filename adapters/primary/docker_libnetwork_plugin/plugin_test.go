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

package docker_libnetwork_plugin_test

import (
	"context"
	"flag"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/primary/docker_libnetwork_plugin"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns/win_networking"
	netSim "github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/simulator"
	"github.com/Juniper/contrail-windows-docker-driver/agent"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"
	"github.com/Microsoft/hcsshim"
	dockerTypes "github.com/docker/docker/api/types"
	dockerTypesContainer "github.com/docker/docker/api/types/container"
	dockerTypesNetwork "github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/docker/go-plugins-helpers/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var netAdapter string
var vswitchName string
var vswitchNameWildcard string
var controllerAddr string
var controllerPort int
var useActualController bool

func init() {
	flag.StringVar(&netAdapter, "netAdapter", "Ethernet0",
		"Network adapter to connect HNS switch to")
	flag.StringVar(&vswitchNameWildcard, "vswitchName", "Layered <adapter>",
		"Name of Transparent virtual switch. Special wildcard \"<adapter>\" will be interpretted "+
			"as value of netAdapter parameter. For example, if netAdapter is \"Ethernet0\", then "+
			"vswitchName will equal \"Layered Ethernet0\". You can use Get-VMSwitch PowerShell "+
			"command to check how the switch is called on your version of OS.")

	flag.StringVar(&controllerAddr, "controllerAddr",
		"10.7.0.54", "Contrail controller addr")
	flag.IntVar(&controllerPort, "controllerPort", 8082, "Contrail controller port")
	flag.BoolVar(&useActualController, "useActualController", true,
		"Whether to use mocked controller or actual.")

	log.SetLevel(log.DebugLevel)
}

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("driver_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Contrail Network Driver test suite",
		[]Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	// Code disabled: cannot mark 'BeforeSuite' block as Pending...
	// vswitchName = strings.Replace(vswitchNameWildcard, "<adapter>", netAdapter, -1)
	// cleanupAll()
})

var _ = AfterSuite(func() {
	// Code disabled: cannot mark 'BeforeSuite' block as Pending...
	// cleanupAll()
})

func cleanupAll() {
	err := common.RestartDocker()
	Expect(err).ToNot(HaveOccurred())
	err = common.HardResetHNS()
	Expect(err).ToNot(HaveOccurred())
	err = win_networking.WaitForValidIPReacquisition(common.AdapterName(netAdapter))
	Expect(err).ToNot(HaveOccurred())

	docker := getDockerClient()
	cleanupAllDockerNetworksAndContainers(docker)
}

func getDockerNetwork(docker *dockerClient.Client, dockerNetID string) (dockerTypes.NetworkResource, error) {
	inspectOptions := dockerTypes.NetworkInspectOptions{
		Scope:   "",
		Verbose: false,
	}
	return docker.NetworkInspect(context.Background(), dockerNetID, inspectOptions)
}

var fakeVRouter ports.VRouter
var contrailController ports.Controller
var server *docker_libnetwork_plugin.DockerPluginServer
var localContrailNetworksRepo ports.LocalContrailNetworkRepository
var project *types.Project

const (
	tenantName  = "agatka"
	networkName = "test_net"
	subnetCIDR  = "1.2.3.4/24"
	defaultGW   = "1.2.3.1"
	timeout     = time.Second * 5
)

type OneTimeListener struct {
	net.Listener
	Received chan (interface{})
}

var _ = Describe("Contrail Network Driver", func() {

	BeforeEach(func() {
		fakeVRouter, server, contrailController, localContrailNetworksRepo, project = newIntegrationModulesUnderTest()
	})
	AfterEach(func() {
		if server.IsServing {
			err := server.StopServing()
			Expect(err).ToNot(HaveOccurred())
		}

		//cleanupAll()
	})

	PIt("can start and stop listening on a named pipe", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		conn, err := sockets.DialPipe(server.PipeAddr, timeout)
		Expect(err).ToNot(HaveOccurred())
		if conn != nil {
			conn.Close()
		}

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		conn, err = sockets.DialPipe(server.PipeAddr, timeout)
		Expect(err).To(HaveOccurred())
		if conn != nil {
			conn.Close()
		}
	})

	PIt("creates a spec file for duration of listening", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(common.PluginSpecFilePath())
		Expect(os.IsNotExist(err)).To(BeFalse())

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(common.PluginSpecFilePath())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	PSpecify("stopping pipe listener won't cause docker restart to fail", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		// make sure docker knows about our driver by creating a network
		_ = createTestContrailNetwork(contrailController)
		docker := getDockerClient()
		_ = createValidDockerNetwork(docker)

		// we need to cleanup here, because otherwise docker keeps the named pipe file open,
		// so we can't remove it
		cleanupAllDockerNetworksAndContainers(docker)

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		err = common.RestartDocker()
		Expect(err).ToNot(HaveOccurred())
	})

	PSpecify("creating vswitch for forwarding extension works", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		network, err := hns.GetHNSNetworkByName(common.RootNetworkName)
		Expect(err).ToNot(HaveOccurred())
		Expect(network).ToNot(BeNil())

		Expect(network.Name).To(Equal(common.RootNetworkName))

		By("root network is not deleted upon shutdown of driver")
		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		_, err = hns.GetHNSNetworkByName(common.RootNetworkName)
		Expect(err).ToNot(HaveOccurred())

		By("if root network exists upon driver startup, additional one is not created")
		netsBefore, err := hns.ListHNSNetworks()
		Expect(err).ToNot(HaveOccurred())

		err = server.StartServing()
		Expect(err).ToNot(HaveOccurred())
		_, err = hns.GetHNSNetworkByName(common.RootNetworkName)
		Expect(err).ToNot(HaveOccurred())

		netsAfter, err := hns.ListHNSNetworks()
		Expect(err).ToNot(HaveOccurred())

		Expect(len(netsBefore)).To(Equal(len(netsAfter)))

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())
	})
})

// TODO: This Describe block is probably to be removed. The only thing we want to test here is json
// marshalling of docker requests probably. Handlers logic is going to be tested in Core, after all.
var _ = PDescribe("On requests from docker daemon", func() {

	var docker *dockerClient.Client

	BeforeEach(func() {
		fakeVRouter, server, contrailController, localContrailNetworksRepo, project = newIntegrationModulesUnderTest()

		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		docker = getDockerClient()
	})
	AfterEach(func() {
		cleanupAllDockerNetworksAndContainers(docker)
		err := common.RestartDocker()
		Expect(err).ToNot(HaveOccurred())

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		err = common.HardResetHNS()
		Expect(err).ToNot(HaveOccurred())

		err = win_networking.WaitForValidIPReacquisition(common.AdapterName(netAdapter))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("on GetCapabilities request", func() {
		It("returns local scope CapabilitiesResponse, nil", func() {
			resp, err := server.GetCapabilities()
			Expect(resp).To(Equal(&network.CapabilitiesResponse{Scope: "local"}))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("on CreateNetwork request", func() {

		var req *network.CreateNetworkRequest
		var genericOptions map[string]interface{}

		BeforeEach(func() {
			ipamData := []*network.IPAMData{
				{
					Pool: subnetCIDR,
				},
			}
			req = &network.CreateNetworkRequest{
				NetworkID: "MyAwesomeNet",
				Options:   make(map[string]interface{}),
				IPv4Data:  ipamData,
			}
			genericOptions = make(map[string]interface{})
		})
		type TestCase struct {
			tenant  string
			network string
		}
		DescribeTable("with different, invalid options",
			func(t TestCase) {
				if t.tenant != "" {
					genericOptions["tenant"] = t.tenant
				}
				if t.network != "" {
					genericOptions["network"] = t.network
				}
				req.Options["com.docker.network.generic"] = genericOptions
				err := server.CreateNetwork(req)
				Expect(err).To(HaveOccurred())
			},
			Entry("subnet doesn't exist in Contrail", TestCase{
				tenant:  tenantName,
				network: "nonexistingNetwork",
			}),
			Entry("tenant doesn't exist in Contrail", TestCase{
				tenant:  "nonexistingTenant",
				network: networkName,
			}),
			Entry("tenant is not specified in request Options", TestCase{
				network: networkName,
			}),
			Entry("network is not specified in request Options", TestCase{
				tenant: tenantName,
			}),
		)

		Context("tenant and subnet exist in Contrail", func() {
			BeforeEach(func() {
				_ = createTestContrailNetwork(contrailController)

				genericOptions["network"] = networkName
				genericOptions["tenant"] = tenantName
				req.Options["com.docker.network.generic"] = genericOptions
			})
			It("responds with nil", func() {
				err := server.CreateNetwork(req)
				Expect(err).ToNot(HaveOccurred())
			})
			It("creates a HNS network", func() {
				netsBefore, err := hns.ListHNSNetworks()
				Expect(err).ToNot(HaveOccurred())

				err = server.CreateNetwork(req)
				Expect(err).ToNot(HaveOccurred())

				netsAfter, err := hns.ListHNSNetworks()
				Expect(err).ToNot(HaveOccurred())
				Expect(netsBefore).To(HaveLen(len(netsAfter) - 1))
			})
		})
	})

	Context("on AllocateNetwork request", func() {
		It("responds with not implemented error", func() {
			req := network.AllocateNetworkRequest{}
			_, err := server.AllocateNetwork(&req)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("on DeleteNetwork request", func() {

		dockerNetID := ""
		var contrailNet *types.VirtualNetwork

		assertRemovesHNSNet := func() {
			resp, err := localContrailNetworksRepo.GetNetwork(tenantName, networkName,
				subnetCIDR)
			Expect(err).To(HaveOccurred())
			Expect(resp).To(BeNil())
		}
		assertRemovesDockerNet := func() {
			_, err := getDockerNetwork(docker, dockerNetID)
			Expect(err).To(HaveOccurred())
		}
		assertDoesNotRemoveContrailNet := func() {
			// TODO: This test must be rewritten using Controller interface.
			// In other words, it mustn't access ApiClient field.

			// 	net, err := types.VirtualNetworkByName(contrailController.ApiClient,
			// 		fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName,
			// 			networkName))
			// 	Expect(err).ToNot(HaveOccurred())
			// 	Expect(net).ToNot(BeNil())
		}

		BeforeEach(func() {
			contrailNet = createTestContrailNetwork(contrailController)
			dockerNetID = createValidDockerNetwork(docker)
		})

		Context("docker, Contrail and HNS networks are fine", func() {
			BeforeEach(func() {
				err := removeDockerNetwork(docker, dockerNetID)
				Expect(err).ToNot(HaveOccurred())
			})
			It("removes HNS net", assertRemovesHNSNet)
			It("removes docker net", assertRemovesDockerNet)
			PIt("does not remove Contrail net", assertDoesNotRemoveContrailNet)
		})

		Context("HNS network doesn't exist", func() {
			// for example, HNS was hard-reset while docker wasn't.
			BeforeEach(func() {
				localContrailNetworksRepo.DeleteNetwork(tenantName, networkName, subnetCIDR)
				err := removeDockerNetwork(docker, dockerNetID)
				Expect(err).ToNot(HaveOccurred())
			})
			It("removes docker net", assertRemovesDockerNet)
			PIt("does not remove Contrail net", assertDoesNotRemoveContrailNet)
		})

		Context("Contrail network doesn't exist", func() {
			// for example, somebody deleted Contrail network before removing HNS network
			BeforeEach(func() {
				err := contrailController.DeleteElementRecursive(contrailNet)
				Expect(err).ToNot(HaveOccurred())
				err = removeDockerNetwork(docker, dockerNetID)
				Expect(err).ToNot(HaveOccurred())
			})
			It("removes HNS net", assertRemovesHNSNet)
			It("removes docker net", assertRemovesDockerNet)
		})
	})

	Context("on FreeNetwork request", func() {
		It("responds with not implemented error", func() {
			req := network.FreeNetworkRequest{}
			err := server.FreeNetwork(&req)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("on CreateEndpoint request", func() {

		Context("Contrail, docker and HNS networks exist", func() {

			containerID := ""
			dockerNetID := ""

			var mockAgentListener *OneTimeListener

			BeforeEach(func() {
				mockAgentListener = startMockAgentListener()
				_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
			})
			AfterEach(func(done Done) {
				// Done channel is ginkgo feature for setting timeouts
				// https://onsi.github.io/ginkgo/#asynchronous-tests
				log.Debugln("Waiting for request")
				<-mockAgentListener.Received
				log.Debugln("Done waiting for request")

				mockAgentListener.Close()
				mockAgentListener = nil
				close(done)
			})
			PIt("allocates Contrail resources", func() {
				// TODO: This test must be rewritten using Controller interface.
				// In other words, it mustn't access ApiClient field.

				// 	net, err := types.VirtualNetworkByName(contrailController.ApiClient,
				// 		fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, networkName))
				// 	Expect(err).ToNot(HaveOccurred())
				// 	Expect(net).ToNot(BeNil())

				// 	// TODO JW-187. For now, VM name is the same as Endpoint ID, not
				// 	// Container ID
				// 	dockerNet, err := getDockerNetwork(docker, dockerNetID)
				// 	Expect(err).ToNot(HaveOccurred())
				// 	vmName := dockerNet.Containers[containerID].EndpointID

				// 	inst, err := types.VirtualMachineByName(contrailController.ApiClient, vmName)
				// 	Expect(err).ToNot(HaveOccurred())
				// 	Expect(inst).ToNot(BeNil())

				// 	vifFQName := fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, vmName)
				// 	vif, err := types.VirtualMachineInterfaceByName(contrailController.ApiClient,
				// 		vifFQName)
				// 	Expect(err).ToNot(HaveOccurred())
				// 	Expect(vif).ToNot(BeNil())

				// 	ip, err := types.InstanceIpByName(contrailController.ApiClient, vif.GetName())
				// 	Expect(err).ToNot(HaveOccurred())
				// 	Expect(ip).ToNot(BeNil())

				// 	ipams, err := net.GetNetworkIpamRefs()
				// 	Expect(err).ToNot(HaveOccurred())
				// 	subnets := ipams[0].Attr.(types.VnSubnetsType).IpamSubnets
				// 	gw := subnets[0].DefaultGateway
				// 	Expect(gw).ToNot(Equal(""))

				// 	macs := vif.GetVirtualMachineInterfaceMacAddresses()
				// 	Expect(macs.MacAddress).To(HaveLen(1))
			})
			It("configures HNS endpoint", func() {
				resp, err := docker.ContainerInspect(context.Background(), containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				ip := resp.NetworkSettings.Networks[networkName].IPAddress
				mac := resp.NetworkSettings.Networks[networkName].MacAddress
				gw := resp.NetworkSettings.Networks[networkName].Gateway

				ep, _ := getTheOnlyHNSEndpoint(server)
				Expect(ep.IPAddress.String()).To(Equal(ip))
				formattedMac := strings.Replace(strings.ToUpper(mac), ":", "-", -1)
				Expect(ep.MacAddress).To(Equal(formattedMac))
				Expect(ep.GatewayAddress).To(Equal(gw))
			})
		})

		Context("Contrail and docker networks exists, HNS network doesn't", func() {
			// for example, HNS was hard-reset while docker wasn't.
			containerID := ""

			BeforeEach(func() {
				_ = createTestContrailNetwork(contrailController)
				_ = createValidDockerNetwork(docker)

				localContrailNetworksRepo.DeleteNetwork(tenantName, networkName, subnetCIDR)
			})
			It("responds with err", func() {
				var err error
				containerID, err = runDockerContainer(docker)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Contrail network exists, docker network doesn't", func() {
			BeforeEach(func() {
				_ = createTestContrailNetwork(contrailController)
			})
			It("responds with err", func() {
				req := &network.CreateEndpointRequest{
					EndpointID: "somecontainerID",
				}
				_, err := server.CreateEndpoint(req)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	// TODO: This Context is marked as Pending, because BeforeEach block must be
	// rewritten using Controller interface.
	PContext("on DeleteEndpoint request", func() {

		// dockerNetID := ""
		containerID := ""
		hnsEndpointID := ""
		// vmName := ""
		var contrailInst *types.VirtualMachine
		// var contrailVif *types.VirtualMachineInterface
		// var contrailIP *types.InstanceIp
		var mockAgentListener *OneTimeListener

		BeforeEach(func() {
			// TODO: This BeforeEach must be rewritten using Controller interface.
			// In other words, it mustn't access ApiClient field.

			// mockAgentListener = startMockAgentListener()
			// _, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
			// _, hnsEndpointID = getTheOnlyHNSEndpoint(server)

			// // TODO JW-187. For now, VM name is the same as Endpoint ID, not
			// // Container ID
			// dockerNet, err := getDockerNetwork(docker, dockerNetID)
			// Expect(err).ToNot(HaveOccurred())
			// vmName = dockerNet.Containers[containerID].EndpointID

			// contrailInst, err = types.VirtualMachineByName(contrailController.ApiClient, vmName)
			// Expect(err).ToNot(HaveOccurred())
			// Expect(contrailInst).ToNot(BeNil())

			// vifFQName := fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, vmName)
			// contrailVif, err = types.VirtualMachineInterfaceByName(contrailController.ApiClient,
			// 	vifFQName)
			// Expect(err).ToNot(HaveOccurred())
			// Expect(contrailVif).ToNot(BeNil())

			// contrailIP, err = types.InstanceIpByName(contrailController.ApiClient, vmName)
			// Expect(err).ToNot(HaveOccurred())
			// Expect(contrailIP).ToNot(BeNil())
		})
		AfterEach(func(done Done) {
			// Done channel is ginkgo feature for setting timeouts
			// https://onsi.github.io/ginkgo/#asynchronous-tests
			log.Debugln("Waiting for request")
			<-mockAgentListener.Received
			log.Debugln("Done waiting for request")

			mockAgentListener.Close()
			mockAgentListener = nil
			close(done)
		})

		assertRemovesDockerEndpoint := func() {
			_, err := docker.ContainerInspect(context.Background(), containerID)
			Expect(err).To(HaveOccurred())
		}

		assertRemovesHNSEndpoint := func() {
			ep, err := hns.GetHNSEndpoint(hnsEndpointID)
			Expect(err).To(HaveOccurred())
			Expect(ep).To(BeNil())
		}

		assertRemovesContrailVM := func() {
			// TODO: This test must be rewritten using Controller interface.
			// In other words, it mustn't access ApiClient field.

			// _, err := types.VirtualMachineByName(contrailController.ApiClient,
			// 	vmName)
			// Expect(err).To(HaveOccurred())

			// _, err = types.VirtualMachineInterfaceByName(contrailController.ApiClient,
			// 	fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, vmName))
			// Expect(err).To(HaveOccPurred())

			// _, err = types.InstanceIpByName(contrailController.ApiClient,
			// 	contrailVif.GetName())
			// Expect(err).To(HaveOccurred())
		}

		Context("happy case: HNS, docker and Contrail states are in sync", func() {
			BeforeEach(func() {
				stopAndRemoveDockerContainer(docker, containerID)
			})
			It("removes docker endpoint", assertRemovesDockerEndpoint)
			It("removes HNS endpoint", assertRemovesHNSEndpoint)
			PIt("removes virtual-machine and its children in Contrail", assertRemovesContrailVM)
		})

		Context("HNS endpoint doesn't exist", func() {
			BeforeEach(func() {
				err := hns.DeleteHNSEndpoint(hnsEndpointID)
				Expect(err).ToNot(HaveOccurred())
				stopAndRemoveDockerContainer(docker, containerID)
			})
			It("removes docker endpoint", assertRemovesDockerEndpoint)
			It("removes virtual-machine and its children in Contrail", assertRemovesContrailVM)
		})

		Context("virtual-machine in Contrail doesn't exist", func() {
			BeforeEach(func() {
				err := contrailController.DeleteElementRecursive(contrailInst)
				Expect(err).ToNot(HaveOccurred())
				stopAndRemoveDockerContainer(docker, containerID)
			})
			It("removes docker endpoint", assertRemovesDockerEndpoint)
			It("removes HNS endpoint", assertRemovesHNSEndpoint)
		})
	})

	Context("on EndpointInfo request", func() {

		dockerNetID := ""
		containerID := ""
		var req *network.InfoRequest

		BeforeEach(func() {
			_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
			dockerNet, err := getDockerNetwork(docker, dockerNetID)
			Expect(err).ToNot(HaveOccurred())
			req = &network.InfoRequest{
				NetworkID:  dockerNetID,
				EndpointID: dockerNet.Containers[containerID].EndpointID,
			}
		})

		Context("queried endpoint exists", func() {
			It("responds with proper InfoResponse", func() {
				resp, err := server.EndpointInfo(req)
				Expect(err).ToNot(HaveOccurred())

				hnsEndpoint, _ := getTheOnlyHNSEndpoint(server)
				Expect(resp.Value).To(HaveKeyWithValue("hnsid", hnsEndpoint.Id))
				Expect(resp.Value).To(HaveKeyWithValue(
					"com.docker.network.endpoint.macaddress", hnsEndpoint.MacAddress))
			})
		})

		Context("queried endpoint doesn't exist", func() {
			BeforeEach(func() {
				deleteTheOnlyHNSEndpoint(server)
			})
			It("responds with err", func() {
				_, err := server.EndpointInfo(req)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("on Join request", func() {

		dockerNetID := ""
		containerID := ""
		var req *network.JoinRequest

		BeforeEach(func() {
			_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
			dockerNet, err := getDockerNetwork(docker, dockerNetID)
			Expect(err).ToNot(HaveOccurred())
			req = &network.JoinRequest{
				NetworkID:  dockerNetID,
				EndpointID: dockerNet.Containers[containerID].EndpointID,
			}
		})

		Context("queried endpoint exists", func() {
			It("responds with valid gateway IP and disabled gw service", func() {
				resp, err := server.Join(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.DisableGatewayService).To(BeTrue())

				contrailNet, err := contrailController.GetNetwork(tenantName, networkName)
				Expect(err).ToNot(HaveOccurred())
				ipams, err := contrailNet.GetNetworkIpamRefs()
				Expect(err).ToNot(HaveOccurred())
				subnets := ipams[0].Attr.(types.VnSubnetsType).IpamSubnets
				contrailGW := subnets[0].DefaultGateway

				Expect(resp.Gateway).To(Equal(contrailGW))
			})
		})

		Context("queried endpoint doesn't exist", func() {
			BeforeEach(func() {
				deleteTheOnlyHNSEndpoint(server)
			})
			It("responds with err", func() {
				_, err := server.Join(req)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("on Leave request", func() {

		dockerNetID := ""
		containerID := ""
		var req *network.LeaveRequest

		BeforeEach(func() {
			_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
			dockerNet, err := getDockerNetwork(docker, dockerNetID)
			Expect(err).ToNot(HaveOccurred())
			req = &network.LeaveRequest{
				NetworkID:  dockerNetID,
				EndpointID: dockerNet.Containers[containerID].EndpointID,
			}
		})

		Context("queried endpoint exists", func() {
			It("responds with nil", func() {
				err := server.Leave(req)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("queried endpoint doesn't exist", func() {
			BeforeEach(func() {
				deleteTheOnlyHNSEndpoint(server)
			})
			It("responds with err", func() {
				err := server.Leave(req)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("on DiscoverNew request", func() {
		It("responds with nil", func() {
			req := network.DiscoveryNotification{}
			err := server.DiscoverNew(&req)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("on DiscoverDelete request", func() {
		It("responds with nil", func() {
			req := network.DiscoveryNotification{}
			err := server.DiscoverDelete(&req)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("on ProgramExternalConnectivity request", func() {
		It("responds with nil", func() {
			req := network.ProgramExternalConnectivityRequest{}
			err := server.ProgramExternalConnectivity(&req)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("on RevokeExternalConnectivity request", func() {
		It("responds with nil", func() {
			req := network.RevokeExternalConnectivityRequest{}
			err := server.RevokeExternalConnectivity(&req)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

// TODO: Remove this function at earliest convenience (one we successfuly extract and rename driver
// Core and all the other adapters).
func newIntegrationModulesUnderTest() (vr ports.VRouter, d *docker_libnetwork_plugin.DockerPluginServer, c ports.Controller, h ports.LocalContrailNetworkRepository, p *types.Project) {
	var err error

	ext := &hyperv_extension.HyperVExtensionSimulator{
		Enabled: false,
		Running: true,
	}
	vr = vrouter.NewHyperVvRouter(ext)

	c = controller_rest.NewFakeControllerAdapter()

	p, err = c.NewProject(common.DomainName, tenantName)
	Expect(err).ToNot(HaveOccurred())

	netRepo := &netSim.InMemContrailNetworksRepository{}
	epRepo := &netSim.InMemEndpointRepository{}
	serverUrl, _ := url.Parse("http://127.0.0.1:9091")
	a := agent.NewAgentRestAPI(http.DefaultClient, serverUrl)

	driverCore, err := driver_core.NewContrailDriverCore(vr, c, a, netRepo, epRepo)
	Expect(err).ToNot(HaveOccurred())
	d = docker_libnetwork_plugin.NewDockerPluginServer(driverCore)

	return
}

// TODO: this function is a duplicate from DriverCore module, but it should probably be removed from
// here in the future, as we will remove such tests.
func createTestContrailNetwork(c ports.Controller) *types.VirtualNetwork {
	network, err := c.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)
	Expect(err).ToNot(HaveOccurred())
	return network
}

func getDockerClient() *dockerClient.Client {
	docker, err := dockerClient.NewEnvClient()
	Expect(err).ToNot(HaveOccurred())
	return docker
}

func runDockerContainer(docker *dockerClient.Client) (string, error) {
	resp, err := docker.ContainerCreate(context.Background(),
		&dockerTypesContainer.Config{
			Image: "microsoft/nanoserver",
		},
		&dockerTypesContainer.HostConfig{
			NetworkMode: networkName,
		},
		nil, "test_container_name")
	Expect(err).ToNot(HaveOccurred())
	containerID := resp.ID
	Expect(containerID).ToNot(Equal(""))

	err = docker.ContainerStart(context.Background(), containerID,
		dockerTypes.ContainerStartOptions{})

	return containerID, err
}

func stopAndRemoveDockerContainer(docker *dockerClient.Client, containerID string) {
	timeout := time.Second * 5
	err := docker.ContainerStop(context.Background(), containerID, &timeout)
	Expect(err).ToNot(HaveOccurred())

	err = docker.ContainerRemove(context.Background(), containerID,
		dockerTypes.ContainerRemoveOptions{Force: true})
	Expect(err).ToNot(HaveOccurred())
}

func createValidDockerNetwork(docker *dockerClient.Client) string {
	return createDockerNetwork(tenantName, networkName, docker)
}

func createDockerNetwork(tenant, network string, docker *dockerClient.Client) string {
	params := &dockerTypes.NetworkCreate{
		Driver: common.DriverName,
		IPAM: &dockerTypesNetwork.IPAM{
			// libnetwork/ipams/windowsipam ("windows") driver is a null ipam driver.
			// We use 0/32 subnet because no preferred address is specified (as documented in
			// source code of windowsipam driver). We do this because our driver has to handle
			// IP assignment.
			// If container has IP before CreateEndpoint request is handled and CreateEndpoint
			// returns a new IP (assigned by Contrail), docker daemon will complain that we cannot
			// reassign IPs. Hence, we tell the IPAM driver to not assign any IPs.
			Driver: "windows",
			Config: []dockerTypesNetwork.IPAMConfig{
				{
					Subnet: "0.0.0.0/32",
				},
			},
		},
		Options: map[string]string{
			"tenant":  tenant,
			"network": network,
		},
	}
	resp, err := docker.NetworkCreate(context.Background(), networkName, *params)
	Expect(err).ToNot(HaveOccurred())
	return resp.ID
}

func removeDockerNetwork(docker *dockerClient.Client, dockerNetID string) error {
	return docker.NetworkRemove(context.Background(), dockerNetID)
}

func cleanupAllDockerNetworksAndContainers(docker *dockerClient.Client) {
	log.Infoln("Cleaning up docker containers")
	containers, err := docker.ContainerList(context.Background(), dockerTypes.ContainerListOptions{All: true})
	Expect(err).ToNot(HaveOccurred())
	for _, c := range containers {
		log.Debugln("Stopping and removing container", c.ID)
		stopAndRemoveDockerContainer(docker, c.ID)
	}
	log.Infoln("Cleaning up docker networks")
	nets, err := docker.NetworkList(context.Background(), dockerTypes.NetworkListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, net := range nets {
		if net.Name == "none" || net.Name == "nat" {
			continue // those networks are pre-defined and cannot be removed (will cause error)
		}
		log.Debugln("Removing docker network", net.Name)
		err = removeDockerNetwork(docker, net.ID)
		Expect(err).ToNot(HaveOccurred())
	}
}

func deleteTheOnlyHNSEndpoint(d *docker_libnetwork_plugin.DockerPluginServer) {
	_, hnsEndpointID := getTheOnlyHNSEndpoint(d)
	err := hns.DeleteHNSEndpoint(hnsEndpointID)
	Expect(err).ToNot(HaveOccurred())
}

func getTheOnlyHNSEndpoint(d *docker_libnetwork_plugin.DockerPluginServer) (*hcsshim.HNSEndpoint, string) {
	hnsNets, err := localContrailNetworksRepo.ListNetworks()
	Expect(err).ToNot(HaveOccurred())
	Expect(hnsNets).To(HaveLen(1))
	eps, err := hns.ListHNSEndpointsOfNetwork(hnsNets[0].Id)
	Expect(err).ToNot(HaveOccurred())
	Expect(eps).To(HaveLen(1))
	hnsEndpointID := eps[0].Id
	hnsEndpoint, err := hns.GetHNSEndpoint(hnsEndpointID)
	Expect(err).ToNot(HaveOccurred())
	Expect(hnsEndpoint).ToNot(BeNil())
	return hnsEndpoint, hnsEndpointID
}

func setupNetworksAndEndpoints(c ports.Controller, docker *dockerClient.Client) (
	*types.VirtualNetwork, string, string) {
	contrailNet := createTestContrailNetwork(c)
	dockerNetID := createValidDockerNetwork(docker)
	containerID, err := runDockerContainer(docker)
	Expect(err).ToNot(HaveOccurred())
	return contrailNet, dockerNetID, containerID
}

func startMockAgentListener() *OneTimeListener {
	listener := OneTimeListener{}
	var err error
	listener.Listener, err = net.Listen("tcp", ":9090") // agent api port
	Expect(err).ToNot(HaveOccurred())
	Expect(listener.Listener).ToNot(BeNil())

	listener.Received = make(chan interface{}, 1)

	go func() {
		conn, err := listener.Accept()
		buf := make([]byte, 2046)
		bytesRead, err := conn.Read(buf)
		if err != nil {
			log.Errorln("Failed to read request", err)
		}
		log.Debugln("Received message:", string(buf[:bytesRead]))
		listener.Received <- 1
		log.Debugln("Sent info about receiveing the request")
		Expect(err).ToNot(HaveOccurred())
		Expect(conn).ToNot(BeNil())
	}()
	return &listener
}
