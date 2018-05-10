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

package configuration

import (
	"flag"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
)

var (
	adapter = flag.String("adapter", "Ethernet0",
		"net adapter for HNS switch, must be physical")
	controllerIP = flag.String("controllerIP", "127.0.0.1",
		"IP address of Contrail Controller API")
	controllerPort = flag.Int("controllerPort", 8082,
		"port of Contrail Controller API")
	agentURL    = flag.String("agentURL", "http://127.0.0.1:9091", "URL of Agent API")
	vswitchName = flag.String("vswitchName", "Layered?<adapter>",
		"Name of Transparent virtual switch. Special wildcard \"<adapter>\" will be interpretted "+
			"as value of netAdapter parameter. For example, if netAdapter is \"Ethernet0\", then "+
			"vswitchName will equal \"Layered Ethernet0\". You can use Get-VMSwitch PowerShell "+
			"command to check how the switch is called on your version of OS.")

	authMethod = flag.String("authMethod", "keystone", "Controller auth method. Specifying it is mandatory. "+
		"(possible values: noauth|keystone)")
	os_auth_url = flag.String("os_auth_url", "", "Keystone auth url. If empty, will read "+
		"from environment variable")
	os_username = flag.String("os_username", "", "Contrail username. If empty, "+
		"will read from environment variable")
	os_tenant_name = flag.String("os_tenant_name", "", "Tenant name. If empty, will read "+
		"environment variable")
	os_password = flag.String("os_password", "", "Contrail password. If empty, will read "+
		"environment variable")
	os_token = flag.String("os_token", "", "Keystone token. If empty, will read "+
		"environment variable")
)

func (conf *Configuration) LoadFromCommandLine() {
	conf.Driver = DriverConf{
		Adapter:        *adapter,
		ControllerIP:   *controllerIP,
		ControllerPort: *controllerPort,
		AgentURL:       *agentURL,
		VSwitchName:    *vswitchName,
	}
	conf.Auth = AuthConf{
		AuthMethod: *authMethod,
		Keystone: auth.KeystoneParams{
			Os_auth_url:    *os_auth_url,
			Os_username:    *os_username,
			Os_tenant_name: *os_tenant_name,
			Os_password:    *os_password,
			Os_token:       *os_token,
		},
	}
}
