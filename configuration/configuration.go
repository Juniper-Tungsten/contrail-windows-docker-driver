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
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
)

type DriverConf struct {
	Adapter        string
	ControllerIP   string
	ControllerPort int
	AgentURL       string
	VSwitchName    string
}

type AuthConf struct {
	AuthMethod string
	Keystone   auth.KeystoneParams
}

type Configuration struct {
	Driver DriverConf
	Auth   AuthConf
}

func NewDefaultConfiguration() (conf Configuration) {
	conf.Driver.Adapter = "Ethernet0"
	conf.Driver.ControllerIP = "192.168.0.10"
	conf.Driver.ControllerPort = 8082
	conf.Driver.AgentURL = "http://127.0.0.1:9091"
	conf.Driver.VSwitchName = "Layered <adapter>"

	conf.Auth.AuthMethod = "noauth"

	conf.Auth.Keystone.Os_auth_url = ""
	conf.Auth.Keystone.Os_username = ""
	conf.Auth.Keystone.Os_tenant_name = ""
	conf.Auth.Keystone.Os_password = ""
	conf.Auth.Keystone.Os_token = ""

	return
}

func DefaultConfigFilepath() string {
	return ""
	// TODO: uncomment the following lines once templating logic is implemented in ansible.
	//       For now, return empty string, so that docker driver defaults to reading only command
	//       line parameters.
	// return string(filepath.Join(os.Getenv("ProgramData"),
	// 	"Contrail", "etc", "contrail", "cnm-driver.conf"))
}
