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

package hns

import (
	"flag"
	"fmt"
	"net"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/Microsoft/hcsshim"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var netAdapter string
var controllerAddr string
var controllerPort int
var useActualController bool

func init() {
	flag.StringVar(&netAdapter, "netAdapter", "Ethernet0",
		"Network adapter to connect HNS switch to")
	flag.StringVar(&controllerAddr, "controllerAddr",
		"10.7.0.54", "Contrail controller addr")
	flag.IntVar(&controllerPort, "controllerPort", 8082, "Contrail controller port")
	flag.BoolVar(&useActualController, "useActualController", true,
		"Whether to use mocked controller or actual.")

	log.SetLevel(log.DebugLevel)
}

func TestHNS(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("hns_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "HNS wrapper test suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	err := common.HardResetHNS()
	Expect(err).ToNot(HaveOccurred())
	err = common.WaitForInterface(common.AdapterName(netAdapter))
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := common.HardResetHNS()
	Expect(err).ToNot(HaveOccurred())
	err = common.WaitForInterface(common.AdapterName(netAdapter))
	Expect(err).ToNot(HaveOccurred())
})

const (
	tenantName  = "agatka"
	networkName = "test_net"
	subnetCIDR  = "10.0.0.0/24"
	defaultGW   = "10.0.0.1"
)

var _ = Describe("HNS wrapper", func() {

	var originalNumNetworks int

	BeforeEach(func() {
		nets, err := ListHNSNetworks()
		Expect(err).ToNot(HaveOccurred())
		originalNumNetworks = len(nets)
	})

	Context("HNS network exists", func() {

		testNetName := "TestNetwork"
		testHnsNetID := ""

		BeforeEach(func() {
			expectNumberOfEndpoints(0)

			Expect(testHnsNetID).To(Equal(""))
			testHnsNetID = MockHNSNetwork(common.AdapterName(netAdapter), testNetName, subnetCIDR,
				defaultGW)
			Expect(testHnsNetID).ToNot(Equal(""))

			net, err := GetHNSNetwork(testHnsNetID)
			Expect(err).ToNot(HaveOccurred())
			Expect(net).ToNot(BeNil())
		})

		AfterEach(func() {
			endpoints, err := ListHNSEndpoints()
			Expect(err).ToNot(HaveOccurred())
			if len(endpoints) > 0 {
				// Cleanup lingering endpoints.
				for _, ep := range endpoints {
					err = DeleteHNSEndpoint(ep.Id)
					Expect(err).ToNot(HaveOccurred())
				}
				expectNumberOfEndpoints(0)
			}

			Expect(testHnsNetID).ToNot(Equal(""))
			err = DeleteHNSNetwork(testHnsNetID)
			Expect(err).ToNot(HaveOccurred())
			_, err = GetHNSNetwork(testHnsNetID)
			Expect(err).To(HaveOccurred())
			testHnsNetID = ""
			nets, err := ListHNSNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).ToNot(BeNil())
			Expect(len(nets)).To(Equal(originalNumNetworks))
		})

		Specify("listing all HNS networks works", func() {
			nets, err := ListHNSNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).ToNot(BeNil())
			Expect(len(nets)).To(Equal(originalNumNetworks + 1))
			found := false
			for _, n := range nets {
				if n.Id == testHnsNetID {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		Specify("getting a single HNS network works", func() {
			net, err := GetHNSNetwork(testHnsNetID)
			Expect(err).ToNot(HaveOccurred())
			Expect(net).ToNot(BeNil())
			Expect(net.Id).To(Equal(testHnsNetID))
		})

		Specify("getting a single HNS network by name works", func() {
			net, err := GetHNSNetworkByName(testNetName)
			Expect(err).ToNot(HaveOccurred())
			Expect(net).ToNot(BeNil())
			Expect(net.Id).To(Equal(testHnsNetID))
		})

		Specify("HNS endpoint operations work", func() {
			hnsEndpointConfig := &hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				Name:           "ep_name",
			}

			endpointID, err := CreateHNSEndpoint(hnsEndpointConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(endpointID).ToNot(Equal(""))

			endpoint, err := GetHNSEndpoint(endpointID)
			Expect(err).ToNot(HaveOccurred())
			Expect(endpoint).ToNot(BeNil())

			expectNumberOfEndpoints(1)

			log.Infoln(endpoint)

			err = DeleteHNSEndpoint(endpointID)
			Expect(err).ToNot(HaveOccurred())

			endpoint, err = GetHNSEndpoint(endpointID)
			Expect(err).To(HaveOccurred())
			Expect(endpoint).To(BeNil())
		})

		Specify("Listing HNS endpoints works", func() {
			hnsEndpointConfig := &hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
			}

			endpointsList, err := ListHNSEndpoints()
			Expect(err).ToNot(HaveOccurred())
			numEndpointsOriginal := len(endpointsList)

			var endpoints [2]string
			for i := 0; i < 2; i++ {
				endpoints[i], err = CreateHNSEndpoint(hnsEndpointConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(endpoints[i]).ToNot(Equal(""))
			}

			expectNumberOfEndpoints(numEndpointsOriginal + 2)

			for _, ep := range endpoints {
				err = DeleteHNSEndpoint(ep)
				Expect(err).ToNot(HaveOccurred())
			}

			expectNumberOfEndpoints(numEndpointsOriginal)
		})

		Specify("Getting HNS endpoint by name works", func() {
			names := []string{"name1", "name2", "name3"}
			for _, name := range names {
				hnsEndpointConfig := &hcsshim.HNSEndpoint{
					VirtualNetwork: testHnsNetID,
					Name:           name,
				}
				_, err := CreateHNSEndpoint(hnsEndpointConfig)
				Expect(err).ToNot(HaveOccurred())
			}

			ep, err := GetHNSEndpointByName("name2")
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.Name).To(Equal("name2"))
		})

		Context("There's a second HNS network", func() {
			secondHNSNetID := ""
			BeforeEach(func() {
				secondHNSNetID = MockHNSNetwork(common.AdapterName(netAdapter), "other_net_name",
					subnetCIDR, defaultGW)

			})
			AfterEach(func() {
				err := DeleteHNSNetwork(secondHNSNetID)
				Expect(err).ToNot(HaveOccurred())
			})
			Specify("Listing HNS endpoints of specific network works", func() {
				config1 := &hcsshim.HNSEndpoint{
					VirtualNetwork: testHnsNetID,
				}
				config2 := &hcsshim.HNSEndpoint{
					VirtualNetwork: secondHNSNetID,
				}

				var epsInFirstNet []string
				var epsInSecondNet []string

				// create 3 endpoints in each network
				for i := 0; i < 3; i++ {
					ep1, err := CreateHNSEndpoint(config1)
					Expect(err).ToNot(HaveOccurred())

					epsInFirstNet = append(epsInFirstNet, ep1)

					ep2, err := CreateHNSEndpoint(config2)
					Expect(err).ToNot(HaveOccurred())

					epsInSecondNet = append(epsInSecondNet, ep2)
				}

				foundEpsOfFirstNet, err := ListHNSEndpointsOfNetwork(testHnsNetID)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundEpsOfFirstNet).To(HaveLen(3))
				for _, ep := range foundEpsOfFirstNet {
					Expect(epsInFirstNet).To(ContainElement(ep.Id))
					Expect(epsInSecondNet).ToNot(ContainElement(ep.Id))
				}

				foundEpsOfSecondNet, err := ListHNSEndpointsOfNetwork(secondHNSNetID)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundEpsOfSecondNet).To(HaveLen(3))
				for _, ep := range foundEpsOfSecondNet {
					Expect(epsInSecondNet).To(ContainElement(ep.Id))
					Expect(epsInFirstNet).ToNot(ContainElement(ep.Id))
				}
			})
		})

		Specify("Creating endpoint in same subnet works", func() {
			_, err := CreateHNSEndpoint(&hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				IPAddress:      net.ParseIP("10.0.0.4"),
			})
			Expect(err).ToNot(HaveOccurred())
			expectNumberOfEndpoints(1)
		})

		Specify("Creating endpoint in different subnet fails", func() {
			_, err := CreateHNSEndpoint(&hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				IPAddress:      net.ParseIP("10.1.0.4"),
			})
			Expect(err).To(HaveOccurred())
			expectNumberOfEndpoints(0)
		})

		Specify("Creating two endpoints with same IP works in same subnet fails", func() {
			_, err := CreateHNSEndpoint(&hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				IPAddress:      net.ParseIP("10.0.0.4"),
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = CreateHNSEndpoint(&hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				IPAddress:      net.ParseIP("10.0.0.4"),
			})
			Expect(err).To(HaveOccurred())

			expectNumberOfEndpoints(1)
		})

		type MACTestCase struct {
			MAC        string
			shouldFail bool
		}
		DescribeTable("Creating an endpoint with specific MACs",
			func(t MACTestCase) {
				epID, err := CreateHNSEndpoint(&hcsshim.HNSEndpoint{
					VirtualNetwork: testHnsNetID,
					MacAddress:     t.MAC,
				})
				if t.shouldFail {
					Expect(err).To(HaveOccurred())
					expectNumberOfEndpoints(0)
				} else {
					Expect(err).ToNot(HaveOccurred())
					ep, err := GetHNSEndpoint(epID)
					Expect(err).ToNot(HaveOccurred())
					Expect(ep.MacAddress).To(Equal(t.MAC))
					expectNumberOfEndpoints(1)
				}
			},
			Entry("11-22-33-44-55-66 works", MACTestCase{
				MAC:        "11-22-33-44-55-66",
				shouldFail: false,
			}),
			Entry("AA-BB-CC-DD-EE-FF works", MACTestCase{
				MAC:        "AA-BB-CC-DD-EE-FF",
				shouldFail: false,
			}),
			Entry("XX-YY-11-22-33-44 fails", MACTestCase{
				MAC:        "XX-YY-11-22-33-44",
				shouldFail: true,
			}),
		)

		Specify("Creating multiple endpoints with conflicting MACs works", func() {
			cfg := &hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				MacAddress:     "11-22-33-44-55-66",
			}
			for i := 0; i < 3; i++ {
				_, err := CreateHNSEndpoint(cfg)
				Expect(err).ToNot(HaveOccurred())
			}
			expectNumberOfEndpoints(3)
		})

		Specify("Creating endpoint with name containing special characters works", func() {
			cfg := &hcsshim.HNSEndpoint{
				VirtualNetwork: testHnsNetID,
				Name:           "A:B123/123",
			}
			_, err := CreateHNSEndpoint(cfg)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("HNS network doesn't exist", func() {

		BeforeEach(func() {
			nets, err := ListHNSNetworks()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(nets)).To(Equal(originalNumNetworks))
		})

		AfterEach(func() {
			nets, err := ListHNSNetworks()
			Expect(err).ToNot(HaveOccurred())
			for _, n := range nets {
				if strings.Contains(n.Name, "nat") {
					continue
				}
				err = DeleteHNSNetwork(n.Id)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		Specify("getting single HNS network returns error", func() {
			net, err := GetHNSNetwork("1234abcd")
			Expect(err).To(HaveOccurred())
			Expect(net).To(BeNil())
		})

		Specify("getting single HNS network by name returns nil, nil", func() {
			net, err := GetHNSNetworkByName("asdf")
			Expect(err).To(BeNil())
			Expect(net).To(BeNil())
		})
	})
})

var _ = Describe("HNS race conditions workarounds", func() {

	var targetAddr string
	const (
		numTries = 20
	)

	BeforeEach(func() {
		if !useActualController {
			Skip("useActualController flag is false. Won't perform HNS race conditions test.")
		}

		targetAddr = fmt.Sprintf("%s:%v", controllerAddr, controllerPort)
		err := common.HardResetHNS()
		Expect(err).ToNot(HaveOccurred())
		err = common.WaitForInterface(common.AdapterName(netAdapter))
		Expect(err).ToNot(HaveOccurred())
	})

	Specify("without HNS networks, connections work", func() {
		conn, err := net.Dial("tcp", targetAddr)
		Expect(err).ToNot(HaveOccurred())
		if conn != nil {
			conn.Close()
		}
	})

	Context("subnet is specified in new HNS switch config", func() {

		subnets := []hcsshim.Subnet{
			{
				AddressPrefix:  "10.0.0.0/24",
				GatewayAddress: "10.0.0.1",
			},
		}
		configuration := &hcsshim.HNSNetwork{
			Type:    "transparent",
			Subnets: subnets,
		}

		Specify("connections don't fail just after new HNS network is created/deleted", func() {
			// net.Dial may fail with error:
			// `dial tcp localhost:80: connectex: A socket operation was attempted to an
			// unreachable network.`
			configuration.NetworkAdapterName = netAdapter
			for i := 0; i < numTries; i++ {
				name := fmt.Sprintf("net%v", i)
				configuration.Name = name
				By(fmt.Sprintf("Creating HNS network %s", name))
				netID, err := CreateHNSNetwork(configuration)
				Expect(err).ToNot(HaveOccurred(), name)
				conn, err := net.Dial("tcp", targetAddr)
				Expect(err).ToNot(HaveOccurred(), name)
				if conn != nil {
					conn.Close()
				}

				By(fmt.Sprintf("Deleting HNS network %s", name))
				err = DeleteHNSNetwork(netID)
				Expect(err).ToNot(HaveOccurred(), name)
				conn, err = net.Dial("tcp", targetAddr)
				Expect(err).ToNot(HaveOccurred(), name)
				if conn != nil {
					conn.Close()
				}
			}
		})

		Specify("connections don't fail on subsequent HNS networks", func() {

			var netIDs []string
			configuration.NetworkAdapterName = netAdapter

			for i := 0; i < numTries; i++ {
				name := fmt.Sprintf("net%v", i)
				configuration.Name = name
				By(fmt.Sprintf("Creating HNS network %s", name))
				netID, err := CreateHNSNetwork(configuration)
				Expect(err).ToNot(HaveOccurred(), name)
				netIDs = append(netIDs, netID)
				conn, err := net.Dial("tcp", targetAddr)
				Expect(err).ToNot(HaveOccurred(), name)
				if conn != nil {
					conn.Close()
				}
			}

			for i, netID := range netIDs {
				name := fmt.Sprintf("net%v", i)
				By(fmt.Sprintf("Deleting HNS network %s", name))
				err := DeleteHNSNetwork(netID)
				Expect(err).ToNot(HaveOccurred(), name)
				conn, err := net.Dial("tcp", targetAddr)
				Expect(err).ToNot(HaveOccurred(), name)
				if conn != nil {
					conn.Close()
				}
			}
		})
	})

	Context("subnet is NOT specified in new HNS switch config", func() {

		configuration := &hcsshim.HNSNetwork{
			Type: "transparent",
		}

		Specify("error does not occur when we don't supply a subnet to new network", func() {
			// CreateHNSNetwork may fail with error:
			// `HNS failed with error : Unspecified error`
			configuration.NetworkAdapterName = netAdapter
			for i := 0; i < numTries; i++ {
				name := fmt.Sprintf("net%v", i)
				configuration.Name = name
				By(fmt.Sprintf("Creating HNS network %s", name))
				netID, err := CreateHNSNetwork(configuration)
				Expect(err).ToNot(HaveOccurred(), name)
				hcsshim.HNSNetworkRequest("DELETE", netID, "")
			}
		})
	})
})

func expectNumberOfEndpoints(num int) {
	eps, err := ListHNSEndpoints()
	Expect(err).ToNot(HaveOccurred())
	Expect(eps).To(HaveLen(num))
}
