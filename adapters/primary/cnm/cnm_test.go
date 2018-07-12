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

package cnm_test

import (
	"testing"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/primary/cnm"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/docker/go-plugins-helpers/network"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("driver_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Contrail Network Driver test suite",
		[]Reporter{junitReporter})
}

var server *cnm.ServerCNM

const (
	tenantName  = "agatka"
	networkName = "test_net"
	subnetCIDR  = "1.2.3.4/24"
	defaultGW   = "1.2.3.1"
	timeout     = time.Second * 5
)

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
			server = cnm.NewServerCNM(&nullCore)
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
