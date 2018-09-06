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

// +build integration

package helpers

import (
	"net/http"
	"net/url"

	"github.com/Juniper/contrail-go-api/types"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/primary/cnm"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	netSim "github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/simulator"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/port_association/agent"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/Juniper/contrail-windows-docker-driver/core/ports"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"
	. "github.com/onsi/gomega"
)

const (
	TenantName  = "agatka"
	NetworkName = "test_net"
	SubnetCIDR  = "1.2.3.4/24"
	DefaultGW   = "1.2.3.1"
)

func NewIntegrationModulesUnderTest() (vr ports.VRouter, d *cnm.ServerCNM, c ports.Controller, netRepo ports.LocalContrailNetworkRepository, p *types.Project) {
	var err error

	ext := &hyperv_extension.HyperVExtensionSimulator{
		Enabled: false,
		Running: true,
	}
	vr = vrouter.NewHyperVvRouter(ext)

	c = controller_rest.NewFakeControllerAdapter()

	p, err = c.NewProject(controller_rest.DomainName, TenantName)
	Expect(err).ToNot(HaveOccurred())

	netRepo = &netSim.InMemContrailNetworksRepository{}
	epRepo := &netSim.InMemEndpointRepository{}
	serverUrl, _ := url.Parse("http://127.0.0.1:9091")
	a := agent.NewAgentRestAPI(http.DefaultClient, serverUrl)

	driverCore, err := driver_core.NewContrailDriverCore(vr, c, a, netRepo, epRepo)
	Expect(err).ToNot(HaveOccurred())
	d = cnm.NewServerCNM(driverCore)

	return
}
