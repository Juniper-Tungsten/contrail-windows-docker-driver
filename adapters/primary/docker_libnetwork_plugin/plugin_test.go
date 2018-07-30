// +build integration
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

var _ = Describe("On requests from docker daemon", func() {
	Context("Requests that are handled by core driver logic", func() {
		BeforeEach(func() {
			// TODO: write these tests sometime - maybe when implementing hanlding of
			// security groups and non-default domains?
			// server = newIntegrationModulesUnderTest()
		})
		Context("on CreateNetwork request", func() {
			PIt("TODO: parameter validation tests", func() {})
		})

		Context("on DeleteNetwork request", func() {
			PIt("TODO: parameter validation tests", func() {})
		})

		Context("on CreateEndpoint request", func() {
			PIt("TODO: parameter validation tests", func() {})
		})

		PContext("on DeleteEndpoint request", func() {
			PIt("TODO: parameter validation tests", func() {})
		})
	})

	Context("Requests that core logic does not really car&e about", func() {
		BeforeEach(func() {
			// These functions shouldn't use core logic in any way, so let's just pass an empty
			// structure.
			nullCore := driver_core.ContrailDriverCore{}
			server = docker_libnetwork_plugin.NewDockerPluginServer(&nullCore)
		})
		Context("on GetCapabilities request", func() {
			It("returns local scope CapabilitiesResponse, nil", func() {
				resp, err := server.GetCapabilities()
				Expect(resp).To(Equal(&network.CapabilitiesResponse{Scope: "local"}))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("on AllocateNetwork request", func() {
			It("responds with not implemented error", func() {
				req := network.AllocateNetworkRequest{}
				_, err := server.AllocateNetwork(&req)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("on FreeNetwork request", func() {
			It("responds with not implemented error", func() {
				req := network.FreeNetworkRequest{}
				err := server.FreeNetwork(&req)
				Expect(err).To(HaveOccurred())
			})
		})

		// TODO: EndpointInfo, Join, Leave tests required docker daemon to ran in the past.
		// Refactor when refactoring those requests in core.

		// Context("on EndpointInfo request", func() {

		// 	dockerNetID := ""
		// 	containerID := ""
		// 	var req *network.InfoRequest

		// 	BeforeEach(func() {
		// 		_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
		// 		dockerNet, err := getDockerNetwork(docker, dockerNetID)
		// 		Expect(err).ToNot(HaveOccurred())
		// 		req = &network.InfoRequest{
		// 			NetworkID:  dockerNetID,
		// 			EndpointID: dockerNet.Containers[containerID].EndpointID,
		// 		}
		// 	})

		// 	Context("queried endpoint exists", func() {
		// 		It("responds with proper InfoResponse", func() {
		// 			resp, err := server.EndpointInfo(req)
		// 			Expect(err).ToNot(HaveOccurred())

		// 			hnsEndpoint, _ := getTheOnlyHNSEndpoint(server)
		// 			Expect(resp.Value).To(HaveKeyWithValue("hnsid", hnsEndpoint.Id))
		// 			Expect(resp.Value).To(HaveKeyWithValue(
		// 				"com.docker.network.endpoint.macaddress", hnsEndpoint.MacAddress))
		// 		})
		// 	})

		// 	Context("queried endpoint doesn't exist", func() {
		// 		BeforeEach(func() {
		// 			deleteTheOnlyHNSEndpoint(server)
		// 		})
		// 		It("responds with err", func() {
		// 			_, err := server.EndpointInfo(req)
		// 			Expect(err).To(HaveOccurred())
		// 		})
		// 	})
		// })

		// Context("on Join request", func() {

		// 	dockerNetID := ""
		// 	containerID := ""
		// 	var req *network.JoinRequest

		// 	BeforeEach(func() {
		// 		_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
		// 		dockerNet, err := getDockerNetwork(docker, dockerNetID)
		// 		Expect(err).ToNot(HaveOccurred())
		// 		req = &network.JoinRequest{
		// 			NetworkID:  dockerNetID,
		// 			EndpointID: dockerNet.Containers[containerID].EndpointID,
		// 		}
		// 	})

		// 	Context("queried endpoint exists", func() {
		// 		It("responds with valid gateway IP and disabled gw service", func() {
		// 			resp, err := server.Join(req)
		// 			Expect(err).ToNot(HaveOccurred())
		// 			Expect(resp.DisableGatewayService).To(BeTrue())

		// 			contrailNet, err := contrailController.GetNetwork(tenantName, networkName)
		// 			Expect(err).ToNot(HaveOccurred())
		// 			ipams, err := contrailNet.GetNetworkIpamRefs()
		// 			Expect(err).ToNot(HaveOccurred())
		// 			subnets := ipams[0].Attr.(types.VnSubnetsType).IpamSubnets
		// 			contrailGW := subnets[0].DefaultGateway

		// 			Expect(resp.Gateway).To(Equal(contrailGW))
		// 		})
		// 	})

		// 	Context("queried endpoint doesn't exist", func() {
		// 		BeforeEach(func() {
		// 			deleteTheOnlyHNSEndpoint(server)
		// 		})
		// 		It("responds with err", func() {
		// 			_, err := server.Join(req)
		// 			Expect(err).To(HaveOccurred())
		// 		})
		// 	})
		// })

		// Context("on Leave request", func() {

		// 	dockerNetID := ""
		// 	containerID := ""
		// 	var req *network.LeaveRequest

		// 	BeforeEach(func() {
		// 		_, dockerNetID, containerID = setupNetworksAndEndpoints(contrailController, docker)
		// 		dockerNet, err := getDockerNetwork(docker, dockerNetID)
		// 		Expect(err).ToNot(HaveOccurred())
		// 		req = &network.LeaveRequest{
		// 			NetworkID:  dockerNetID,
		// 			EndpointID: dockerNet.Containers[containerID].EndpointID,
		// 		}
		// 	})

		// 	Context("queried endpoint exists", func() {
		// 		It("responds with nil", func() {
		// 			err := server.Leave(req)
		// 			Expect(err).ToNot(HaveOccurred())
		// 		})
		// 	})

		// 	Context("queried endpoint doesn't exist", func() {
		// 		BeforeEach(func() {
		// 			deleteTheOnlyHNSEndpoint(server)
		// 		})
		// 		It("responds with err", func() {
		// 			err := server.Leave(req)
		// 			Expect(err).To(HaveOccurred())
		// 		})
		// 	})
		// })

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
	// TODO: refactor or remove the relevant tests - I'm not sure if we need to be testing this
	// here. This package should deal with CNM API only.

	// hnsNets, err := localContrailNetworksRepo.ListNetworks()
	// Expect(err).ToNot(HaveOccurred())
	// Expect(hnsNets).To(HaveLen(1))
	// eps, err := hns.ListHNSEndpointsOfNetwork(hnsNets[0].Id)
	// Expect(err).ToNot(HaveOccurred())
	// Expect(eps).To(HaveLen(1))
	// hnsEndpointID := eps[0].Id
	// hnsEndpoint, err := hns.GetHNSEndpoint(hnsEndpointID)
	// Expect(err).ToNot(HaveOccurred())
	// Expect(hnsEndpoint).ToNot(BeNil())
	// return hnsEndpoint, hnsEndpointID
	return nil, ""
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
