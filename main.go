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

package main

import (
	"errors"
	"flag"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/primary/cnm"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/port_association/agent"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"
	"github.com/Juniper/contrail-windows-docker-driver/logging"
	log "github.com/sirupsen/logrus"
)

func main() {

	var adapter = flag.String("adapter", "Ethernet0",
		"net adapter for HNS switch, must be physical")
	var controllerIP = flag.String("controllerIP", "127.0.0.1",
		"IP address of Contrail Controller API")
	var controllerPort = flag.Int("controllerPort", 8082,
		"port of Contrail Controller API")
	var agentURL = flag.String("agentURL", "http://127.0.0.1:9091", "URL of Agent API")
	var logPath = flag.String("logPath", logging.DefaultLogFilepath(), "log filepath")
	var logLevelString = flag.String("logLevel", "Info",
		"log verbosity (possible values: Debug|Info|Warn|Error|Fatal|Panic)")
	var vswitchNameWildcard = flag.String("vswitchName", "Layered?<adapter>",
		"Name of Transparent virtual switch. Special wildcard \"<adapter>\" will be interpretted "+
			"as value of netAdapter parameter. For example, if netAdapter is \"Ethernet0\", then "+
			"vswitchName will equal \"Layered Ethernet0\". You can use Get-VMSwitch PowerShell "+
			"command to check how the switch is called on your version of OS.")
	var os_auth_url = flag.String("os_auth_url", "", "Keystone auth url. If empty, will read "+
		"from environment variable")
	var os_username = flag.String("os_username", "", "Contrail username. If empty, "+
		"will read from environment variable")
	var os_tenant_name = flag.String("os_tenant_name", "", "Tenant name. If empty, will read "+
		"environment variable")
	var os_password = flag.String("os_password", "", "Contrail password. If empty, will read "+
		"environment variable")
	var os_token = flag.String("os_token", "", "Keystone token. If empty, will read "+
		"environment variable")
	var authMethod = flag.String("authMethod", "keystone", "Controller auth method. Specifying it is mandatory. "+
		"(possible values: noauth|keystone)")
	flag.Parse()

	logHook, err := logging.SetupHook(*logPath, *logLevelString)
	if err != nil {
		log.Errorf("Setting up logging failed: %s", err)
		return
	}
	defer logHook.Close()

	vswitchName := strings.Replace(*vswitchNameWildcard, "<adapter>", *adapter, -1)

	keys := &auth.KeystoneParams{
		Os_auth_url:    *os_auth_url,
		Os_username:    *os_username,
		Os_tenant_name: *os_tenant_name,
		Os_password:    *os_password,
		Os_token:       *os_token,
	}
	keys.LoadFromEnvironment()

	hypervExtension := hyperv_extension.NewHyperVvRouterForwardingExtension(vswitchName)
	vrouter := vrouter.NewHyperVvRouter(hypervExtension)

	controller, err := NewControllerAdapter(*authMethod, *controllerIP, *controllerPort, keys)

	if err != nil {
		log.Error(err)
		return
	}

	agentUrl, err := url.Parse(*agentURL)
	if err != nil {
		log.Error(err)
		return
	}

	agent := agent.NewAgentRestAPI(http.DefaultClient, agentUrl)

	netRepo, err := hns.NewHNSContrailNetworksRepository(*adapter)
	if err != nil {
		log.Error(err)
		return
	}

	epRepo := &hns.HNSEndpointRepository{}

	core, err := driver_core.NewContrailDriverCore(vrouter, controller, agent, netRepo, epRepo)
	if err != nil {
		log.Error(err)
		return
	}

	d := cnm.NewServerCNM(core)
	if err = d.StartServing(); err != nil {
		log.Error(err)
		return
	}
	defer d.StopServing()

	waitForSigInt()
}

func waitForSigInt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Infoln("Good bye")
}

func NewControllerAdapter(authMethod, ip string, port int, keys *auth.KeystoneParams) (
	*controller_rest.ControllerAdapter, error) {
	switch authMethod {
	case "keystone":
		return controller_rest.NewControllerWithKeystoneAdapter(keys, ip, port)
	case "noauth":
		return controller_rest.NewControllerInsecureAdapter(ip, port)
	default:
		return nil, errors.New("Unsupported authentication method. Use -authMethod flag.")
	}
}
