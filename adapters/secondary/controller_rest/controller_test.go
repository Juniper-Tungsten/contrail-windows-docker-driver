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

package controller_rest_test

import (
	"flag"
	"fmt"
	"testing"

	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/types"
	log "github.com/sirupsen/logrus"

	. "github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var controllerAddr string
var controllerPort int
var useActualController bool

func init() {
	flag.StringVar(&controllerAddr, "controllerAddr",
		"10.7.0.54", "Contrail controller addr")
	flag.IntVar(&controllerPort, "controllerPort", 8082, "Contrail controller port")
	flag.BoolVar(&useActualController, "useActualController", false,
		"Whether to use mocked controller or actual.")

	log.SetLevel(log.DebugLevel)
}

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("controller_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Controller client test suite",
		[]Reporter{junitReporter})
}

const (
	tenantName   = "agatka"
	networkName  = "test_net"
	subnetCIDR   = "10.10.10.0/24"
	subnetPrefix = "10.10.10.0"
	subnetMask   = 24
	defaultGW    = "10.10.10.1"
	ifaceMac     = "contrail_pls_check_macs"
	containerID  = "12345678901"
)

var _ = BeforeSuite(func() {
	// Code disabled: cannot mark 'BeforeSuite' block as Pending...
	// if useActualController {
	// 	// this cleans up
	// 	client, _ := NewClientAndProject(tenantName, controllerAddr, controllerPort)
	// 	CleanupLingeringVM(client, containerID)
	// }
})

var _ = Describe("ControllerAdapterImpl", func() {

	var client *ControllerAdapterImpl
	var project *types.Project

	BeforeEach(func() {
		if useActualController {
			// TODO: Create actual controller instance, like in main.go
		} else {
			client, project = NewTestClientAndProject(tenantName)
		}
	})

	AfterEach(func() {
		if useActualController {
			CleanupLingeringVM(client, containerID)
		}
	})

	Specify("cleaning up resources that are referred to by two other doesn't fail", func() {
		// instanceIP and VMI are both referred to by virtual network, and instanceIP refers
		// to VMI
		testNetwork := CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
			project)
		testInterface := CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
			containerID)
		_ = CreateTestInstance(client.ApiClient, testInterface, containerID)
		_ = CreateTestInstanceIP(client.ApiClient, tenantName, testInterface,
			testNetwork)

		// shouldn't error when creating new client and project
		if useActualController {
			// TODO: Create actual controller instance, like in main.go
		} else {
			client, project = NewTestClientAndProject(tenantName)
		}
	})

	Specify("recursive deletion removes elements down the ref tree", func() {
		testNetwork := CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
			project)
		testInterface := CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
			containerID)
		testInstance := CreateTestInstance(client.ApiClient, testInterface, containerID)
		testInstanceIP := CreateTestInstanceIP(client.ApiClient, tenantName, testInterface,
			testNetwork)

		err := client.DeleteContainer(containerID)
		Expect(err).ToNot(HaveOccurred())

		_, err = client.ApiClient.FindByUuid(testNetwork.GetType(), testNetwork.GetUuid())
		Expect(err).ToNot(HaveOccurred())

		for _, supposedToBeRemovedObject := range []contrail.IObject{testInstance, testInterface,
			testInstanceIP} {
			_, err = client.ApiClient.FindByUuid(supposedToBeRemovedObject.GetType(),
				supposedToBeRemovedObject.GetUuid())
			Expect(err).To(HaveOccurred())
		}
	})

	Describe("getting Contrail network", func() {
		Context("when network already exists in Contrail", func() {
			var testNetwork *types.VirtualNetwork
			BeforeEach(func() {
				testNetwork = CreateTestNetworkWithSubnet(client.ApiClient, networkName,
					subnetCIDR, project)
			})
			It("returns it", func() {
				net, err := client.GetNetwork(tenantName, networkName)
				Expect(err).ToNot(HaveOccurred())
				Expect(net.GetUuid()).To(Equal(testNetwork.GetUuid()))
			})
		})
		Context("when network doesn't exist in Contrail", func() {
			It("returns an error", func() {
				net, err := client.GetNetwork(tenantName, networkName)
				Expect(err).To(HaveOccurred())
				Expect(net).To(BeNil())
			})
		})
	})

	Describe("creating Contrail network with subnet", func() {
		It("when network with the same subnet already exists in Contrail, returns it", func() {
			testNetwork := CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR, project)

			net, err := client.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)

			Expect(err).ToNot(HaveOccurred())
			Expect(net).ToNot(BeNil())
			Expect(net.GetNetworkIpamRefs()).To(HaveLen(1))
			Expect(net.GetUuid()).To(Equal(testNetwork.GetUuid()))
		})
		It("when network doesn't exist in Contrail, it creates and returns it", func() {
			net, err := client.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)

			Expect(err).ToNot(HaveOccurred())
			Expect(net).ToNot(BeNil())
			Expect(net.GetNetworkIpamRefs()).To(HaveLen(1))
		})
		It("when network with a different subnet already exists in Contrail, returns error", func() {
			otherCIDR := "5.6.7.8/24"
			Expect(subnetCIDR).ToNot(Equal(otherCIDR)) // sanity check
			CreateTestNetworkWithSubnet(client.ApiClient, networkName, otherCIDR, project)

			net, err := client.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)

			Expect(err).To(HaveOccurred())
			Expect(net).To(BeNil())
		})
		It("when network without a subnet already exists in Contrail, returns error", func() {
			CreateTestNetwork(client.ApiClient, networkName, project)

			net, err := client.CreateNetworkWithSubnet(tenantName, networkName, subnetCIDR)

			Expect(err).To(HaveOccurred())
			Expect(net).To(BeNil())
		})
	})

	Describe("getting Contrail subnet info", func() {
		assertGettingSubnetFails := func(getTestedNet func() *types.VirtualNetwork,
			CIDR string) func() {
			return func() {
				_, err := client.GetIpamSubnet(getTestedNet(), CIDR)
				Expect(err).To(HaveOccurred())
			}
		}
		Context("network has one subnet with default gateway", func() {
			var testNetwork *types.VirtualNetwork
			BeforeEach(func() {
				testNetwork = CreateTestNetwork(client.ApiClient, networkName, project)
				AddSubnetWithDefaultGateway(client.ApiClient, subnetPrefix, defaultGW,
					subnetMask, testNetwork)
			})
			Specify("getting subnet meta works", func() {
				ipam, err := client.GetIpamSubnet(testNetwork, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(ipam.DefaultGateway).To(Equal(defaultGW))
				Expect(ipam.Subnet.IpPrefix).To(Equal(subnetPrefix))
				Expect(ipam.Subnet.IpPrefixLen).To(Equal(subnetMask))
			})
			Specify("getting subnet when specifying CIDR works", func() {
				_, err := client.GetIpamSubnet(testNetwork, subnetCIDR)
				Expect(err).ToNot(HaveOccurred())
			})
			Specify("getting subnet when specifying CIDR not in Contrail fails",
				assertGettingSubnetFails(func() *types.VirtualNetwork {
					return testNetwork
				}, "1.2.3.4/16"))
		})
		Context("network has one subnet without default gateway", func() {
			var testNetwork *types.VirtualNetwork
			BeforeEach(func() {
				testNetwork = CreateTestNetworkWithSubnet(client.ApiClient, networkName,
					subnetCIDR, project)
			})
			Specify("getting default gw IP returns error", func() {
				if !useActualController {
					Skip("test fails (pending) when using mocked client")
				}
				ipam, err := client.GetIpamSubnet(testNetwork, "")
				Expect(err).ToNot(HaveOccurred())
				if useActualController {
					Expect(ipam.DefaultGateway).ToNot(Equal(""))
					Expect(err).ToNot(HaveOccurred())
				} else {
					// mocked controller lacks some logic here
					Expect(ipam.DefaultGateway).To(Equal(""))
					Expect(err).To(HaveOccurred())
				}
			})
			Specify("getting subnet prefix and prefix len works", func() {
				ipam, err := client.GetIpamSubnet(testNetwork, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(ipam.Subnet.IpPrefix).To(Equal(subnetPrefix))
				Expect(ipam.Subnet.IpPrefixLen).To(Equal(subnetMask))
			})
		})
		Context("network doesn't have subnets", func() {
			var testNetwork *types.VirtualNetwork
			BeforeEach(func() {
				testNetwork = CreateTestNetwork(client.ApiClient, networkName, project)
			})
			Specify("getting subnet returns error",
				assertGettingSubnetFails(func() *types.VirtualNetwork {
					return testNetwork
				}, ""))
		})
		Context("network has multiple subnets", func() {
			var testNetwork *types.VirtualNetwork
			const (
				prefix1 = "10.10.10.0"
				gw1     = "10.10.10.1"
				cidr1   = "10.10.10.0/24"
				prefix2 = "10.20.20.0"
				gw2     = "10.20.20.1"
				cidr2   = "10.20.20.0/24"
			)
			BeforeEach(func() {
				testNetwork = CreateTestNetwork(client.ApiClient, networkName, project)
				AddSubnetWithDefaultGateway(client.ApiClient, prefix1, gw1, 24,
					testNetwork)
				AddSubnetWithDefaultGateway(client.ApiClient, prefix2, gw2, 24,
					testNetwork)
			})
			Context("user specified valid subnet", func() {
				Specify("getting specific subnets works", func() {
					ipam1, err := client.GetIpamSubnet(testNetwork, cidr1)
					Expect(err).ToNot(HaveOccurred())
					Expect(ipam1.DefaultGateway).To(Equal(gw1))

					ipam2, err := client.GetIpamSubnet(testNetwork, cidr2)
					Expect(err).ToNot(HaveOccurred())
					Expect(ipam2.DefaultGateway).To(Equal(gw2))

					Expect(ipam1.Subnet.IpPrefix).NotTo(Equal(ipam2.Subnet.IpPrefix))
				})
			})
			Context("user didn't specify a subnet", func() {
				Specify("getting subnet1 returns error",
					assertGettingSubnetFails(func() *types.VirtualNetwork {
						return testNetwork
					}, ""))
				Specify("getting subnet2 returns error",
					assertGettingSubnetFails(func() *types.VirtualNetwork {
						return testNetwork
					}, ""))
			})
			Context("user specified invalid subnet", func() {
				Specify("getting subnet1 returns error",
					assertGettingSubnetFails(func() *types.VirtualNetwork {
						return testNetwork
					}, "10.12.13.0/24"))
				Specify("getting subnet2 returns error",
					assertGettingSubnetFails(func() *types.VirtualNetwork {
						return testNetwork
					}, "10.12.13.0/24"))
			})
		})
	})

	Describe("getting or creating Contrail virtual interface", func() {
		var testNetwork *types.VirtualNetwork
		BeforeEach(func() {
			testNetwork = CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
				project)
		})
		Context("when vif already exists in Contrail", func() {
			var testInterface *types.VirtualMachineInterface
			BeforeEach(func() {
				testInterface = CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
					containerID)
			})
			It("returns existing vif", func() {
				iface, err := client.GetOrCreateInterface(testNetwork, tenantName, containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(iface).ToNot(BeNil())
				Expect(iface.GetUuid()).To(Equal(testInterface.GetUuid()))
			})
			It("assigns correct FQName to vif", func() {
				iface, err := client.GetOrCreateInterface(testNetwork, tenantName, containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(iface).ToNot(BeNil())
				Expect(iface.GetFQName()).To(Equal([]string{common.DomainName, tenantName,
					containerID}))
			})
			It("does not change vif security group", func() {
				iface, err := client.GetOrCreateInterface(testNetwork, tenantName, containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(iface).ToNot(BeNil())
				Expect(iface.GetSecurityGroupRefs()).To(BeNil())
				Expect(iface.GetPortSecurityEnabled()).To(BeFalse())
			})
		})
		Context("when vif doesn't exist in Contrail", func() {
			Context("when default security group exists", func() {
				It("creates a new vif", func() {
					iface, err := client.GetOrCreateInterface(testNetwork, tenantName, containerID)
					Expect(err).ToNot(HaveOccurred())
					Expect(iface).ToNot(BeNil())
					existingIface, err := types.VirtualMachineInterfaceByUuid(client.ApiClient,
						iface.GetUuid())
					Expect(err).ToNot(HaveOccurred())
					Expect(existingIface.GetUuid()).To(Equal(iface.GetUuid()))
				})
				It("adds the vif to default security group and enables port security", func() {
					iface, err := client.GetOrCreateInterface(testNetwork, tenantName, containerID)
					Expect(err).ToNot(HaveOccurred())
					Expect(iface).ToNot(BeNil())
					existingIface, err := types.VirtualMachineInterfaceByUuid(client.ApiClient,
						iface.GetUuid())
					Expect(existingIface.GetSecurityGroupRefs()).ToNot(BeNil())
					Expect(existingIface.GetPortSecurityEnabled()).To(BeTrue())
				})
			})
			Context("when default security group doesn't exist", func() {
				BeforeEach(func() {
					RemoveTestSecurityGroup(client.ApiClient, "default", project)
				})
				It("returns an error", func() {
					iface, err := client.GetOrCreateInterface(testNetwork, tenantName, containerID)
					Expect(err).To(HaveOccurred())
					Expect(iface).To(BeNil())
				})
			})
		})
	})

	Describe("getting existing Contrail virtual interface", func() {
		var testNetwork *types.VirtualNetwork
		BeforeEach(func() {
			testNetwork = CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
				project)
		})
		Context("when vif already exists in Contrail", func() {
			It("returns existing vif", func() {
				testInterface := CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
					containerID)

				iface, err := client.GetExistingInterface(testNetwork, tenantName, containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(iface).ToNot(BeNil())
				Expect(iface.GetUuid()).To(Equal(testInterface.GetUuid()))
			})
		})
		Context("when vif doesn't exist in Contrail", func() {
			It("returns error", func() {
				_, err := client.GetExistingInterface(testNetwork, tenantName, containerID)
				Expect(err).To(HaveOccurred())
			})
			It("does not create vif", func() {
				_, _ = client.GetExistingInterface(testNetwork, tenantName, containerID)
				fqName := fmt.Sprintf("%s:%s:%s", common.DomainName, tenantName, containerID)
				_, err := types.VirtualMachineInterfaceByName(client.ApiClient, fqName)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("getting Contrail instance", func() {
		var testInterface *types.VirtualMachineInterface
		BeforeEach(func() {
			testNetwork := CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
				project)
			testInterface = CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
				containerID)
		})
		Context("when instance already exists in Contrail", func() {
			var testInstance *types.VirtualMachine
			BeforeEach(func() {
				testInstance = CreateTestInstance(client.ApiClient, testInterface, containerID)
			})
			It("returns existing instance", func() {
				instance, err := client.GetOrCreateInstance(testInterface, containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance).ToNot(BeNil())
				Expect(instance.GetUuid()).To(Equal(testInstance.GetUuid()))
			})
		})
		Context("when instance doesn't exist in Contrail", func() {
			It("creates a new instance", func() {
				instance, err := client.GetOrCreateInstance(testInterface, containerID)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance).ToNot(BeNil())

				existingInst, err := types.VirtualMachineByUuid(client.ApiClient,
					instance.GetUuid())
				Expect(err).ToNot(HaveOccurred())
				Expect(existingInst.GetUuid()).To(Equal(instance.GetUuid()))
			})
		})
	})

	Describe("getting virtual interface MAC", func() {
		var testInterface *types.VirtualMachineInterface
		BeforeEach(func() {
			testNetwork := CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
				project)
			testInterface = CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
				containerID)
		})
		Context("when vif has a VM", func() {
			BeforeEach(func() {
				_ = CreateTestInstance(client.ApiClient, testInterface, containerID)
			})
			It("returns MAC address", func() {
				if !useActualController {
					Skip("test fails (pending) when using mocked client")
				}
				mac, err := client.GetInterfaceMac(testInterface)
				Expect(err).ToNot(HaveOccurred())
				Expect(mac).ToNot(Equal("")) // dunno how to get actual MAC when given Instance
			})
		})
		Context("when vif has MAC", func() {
			BeforeEach(func() {
				AddMacToInterface(client.ApiClient, ifaceMac, testInterface)
			})
			It("returns MAC address", func() {
				mac, err := client.GetInterfaceMac(testInterface)
				Expect(err).ToNot(HaveOccurred())
				Expect(mac).To(Equal(ifaceMac))
			})
		})
		Context("when vif doesn't have a MAC", func() {
			It("returns error", func() {
				mac, err := client.GetInterfaceMac(testInterface)
				Expect(err).To(HaveOccurred())
				Expect(mac).To(Equal(""))
			})
		})
	})

	Describe("getting Contrail instance IP", func() {
		var testNetwork *types.VirtualNetwork
		var testInstance *types.VirtualMachine
		var testInterface *types.VirtualMachineInterface
		BeforeEach(func() {
			testNetwork = CreateTestNetworkWithSubnet(client.ApiClient, networkName, subnetCIDR,
				project)
			testInterface = CreateMockedInterface(client.ApiClient, testNetwork, tenantName,
				containerID)
			testInstance = CreateTestInstance(client.ApiClient, testInterface, containerID)
		})
		Context("when instance IP already exists in Contrail", func() {
			var testInstanceIP *types.InstanceIp
			BeforeEach(func() {
				testInstanceIP = CreateTestInstanceIP(client.ApiClient, tenantName,
					testInterface, testNetwork)
			})
			It("returns existing instance IP", func() {
				instanceIP, err := client.GetOrCreateInstanceIp(testNetwork, testInterface, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(instanceIP).ToNot(BeNil())
				Expect(instanceIP.GetUuid()).To(Equal(testInstanceIP.GetUuid()))
				Expect(instanceIP.GetInstanceIpAddress()).To(Equal(
					testInstanceIP.GetInstanceIpAddress()))

				existingIP, err := types.InstanceIpByUuid(client.ApiClient, instanceIP.GetUuid())
				Expect(err).ToNot(HaveOccurred())
				Expect(existingIP.GetUuid()).To(Equal(instanceIP.GetUuid()))
			})
		})
		Context("when instance IP doesn't exist in Contrail", func() {
			It("creates new instance IP", func() {
				instanceIP, err := client.GetOrCreateInstanceIp(testNetwork, testInterface, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(instanceIP).ToNot(BeNil())
				Expect(instanceIP.GetInstanceIpAddress()).ToNot(Equal(""))

				existingIP, err := types.InstanceIpByUuid(client.ApiClient, instanceIP.GetUuid())
				Expect(err).ToNot(HaveOccurred())
				Expect(existingIP.GetUuid()).To(Equal(instanceIP.GetUuid()))
			})
		})
	})
})
