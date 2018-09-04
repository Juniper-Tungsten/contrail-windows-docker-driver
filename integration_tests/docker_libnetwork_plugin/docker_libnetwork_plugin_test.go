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

package cnm_integration_test

import (
	"context"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/primary/cnm"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns/win_networking"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	"github.com/Juniper/contrail-windows-docker-driver/integration_tests/helpers"
	sockets "github.com/Microsoft/go-winio"
	dockerTypes "github.com/docker/docker/api/types"
	dockerTypesContainer "github.com/docker/docker/api/types/container"
	dockerTypesNetwork "github.com/docker/docker/api/types/network"
	dockerClient "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var netAdapter string
var vswitchName string
var vswitchNameWildcard string

func init() {
	flag.StringVar(&netAdapter, "netAdapter", "Ethernet0",
		"Network adapter to connect HNS switch to")
	flag.StringVar(&vswitchNameWildcard, "vswitchName", "Layered <adapter>",
		"Name of Transparent virtual switch. Special wildcard \"<adapter>\" will be interpretted "+
			"as value of netAdapter parameter. For example, if netAdapter is \"Ethernet0\", then "+
			"vswitchName will equal \"Layered Ethernet0\". You can use Get-VMSwitch PowerShell "+
			"command to check how the switch is called on your version of OS.")

	log.SetLevel(log.DebugLevel)
	vswitchName = strings.Replace(vswitchNameWildcard, "<adapter>", netAdapter, -1)
}

func TestDockerPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("DockerPlugin_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "DockerPlugin wrapper test suite", []Reporter{junitReporter})
}

// NOTE: these tests were not ran in quite a while... sorry.
var _ = Describe("Contrail Docker Libnetwork Plugin registering and listening", func() {
	var fakeVRouter ports.VRouter
	var contrailController ports.Controller
	var server *cnm.ServerCNM
	var localContrailNetworksRepo ports.LocalContrailNetworkRepository
	var project *types.Project

	BeforeEach(func() {
		fakeVRouter, server, contrailController, localContrailNetworksRepo, project = helpers.NewIntegrationModulesUnderTest()
	})
	AfterEach(func() {
		if server.IsServing {
			err := server.StopServing()
			Expect(err).ToNot(HaveOccurred())
		}

		By("cleanup after test")
		cleanupAll()
	})

	PIt("can start and stop listening on a named pipe", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		timeout := time.Second * 5
		conn, err := sockets.DialPipe(server.PipeAddr, &timeout)
		Expect(err).ToNot(HaveOccurred())
		if conn != nil {
			conn.Close()
		}

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		conn, err = sockets.DialPipe(server.PipeAddr, &timeout)
		Expect(err).To(HaveOccurred())
		if conn != nil {
			conn.Close()
		}
	})

	PIt("creates a spec file for duration of listening", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(server.PluginSpecFilePath())
		Expect(os.IsNotExist(err)).To(BeFalse())

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(server.PluginSpecFilePath())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	PSpecify("stopping pipe listener won't cause docker restart to fail", func() {
		err := server.StartServing()
		Expect(err).ToNot(HaveOccurred())

		By("make sure docker knows about our driver by creating a network")
		_ = createTestContrailNetwork(contrailController)
		docker := getDockerClient()
		_ = createValidDockerNetwork(docker)

		By("we need to cleanup here, because otherwise docker keeps the named pipe file open, so we can't remove it")
		cleanupAllDockerNetworksAndContainers(docker)

		err = server.StopServing()
		Expect(err).ToNot(HaveOccurred())

		err = helpers.RestartDocker()
		Expect(err).ToNot(HaveOccurred())
	})
})

//
//
// DOCKER HELPERS
//
//

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
			NetworkMode: helpers.NetworkName,
		},
		nil, "test_container_name")
	Expect(err).ToNot(HaveOccurred())
	containerID := resp.ID
	Expect(containerID).ToNot(Equal(""))

	err = docker.ContainerStart(context.Background(), containerID,
		dockerTypes.ContainerStartOptions{})

	return containerID, err
}

func setupNetworksAndEndpoints(c ports.Controller, docker *dockerClient.Client) (
	*types.VirtualNetwork, string, string) {
	contrailNet := createTestContrailNetwork(c)
	dockerNetID := createValidDockerNetwork(docker)
	containerID, err := runDockerContainer(docker)
	Expect(err).ToNot(HaveOccurred())
	return contrailNet, dockerNetID, containerID
}

func createValidDockerNetwork(docker *dockerClient.Client) string {
	return createDockerNetwork(helpers.TenantName, helpers.NetworkName, docker)
}

func createDockerNetwork(tenant, network string, docker *dockerClient.Client) string {
	params := &dockerTypes.NetworkCreate{
		Driver: cnm.DriverName,
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
	resp, err := docker.NetworkCreate(context.Background(), helpers.NetworkName, *params)
	Expect(err).ToNot(HaveOccurred())
	return resp.ID
}

func cleanupAll() {
	err := helpers.RestartDocker()
	Expect(err).ToNot(HaveOccurred())
	err = helpers.HardResetHNS()
	Expect(err).ToNot(HaveOccurred())
	err = win_networking.WaitForValidIPReacquisition(netAdapter)
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

func stopAndRemoveDockerContainer(docker *dockerClient.Client, containerID string) {
	timeout := time.Second * 5
	err := docker.ContainerStop(context.Background(), containerID, &timeout)
	Expect(err).ToNot(HaveOccurred())

	err = docker.ContainerRemove(context.Background(), containerID,
		dockerTypes.ContainerRemoveOptions{Force: true})
	Expect(err).ToNot(HaveOccurred())
}

func createTestContrailNetwork(c ports.Controller) *types.VirtualNetwork {
	network, err := c.CreateNetworkWithSubnet(helpers.TenantName, helpers.NetworkName, helpers.SubnetCIDR)
	Expect(err).ToNot(HaveOccurred())
	return network
}
