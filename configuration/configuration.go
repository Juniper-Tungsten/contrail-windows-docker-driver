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
	"errors"
	"os"
	"path/filepath"

	"github.com/Juniper/contrail-windows-docker-driver/logging"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
)

type DriverConf struct {
	Adapter        string
	ControllerIP   string
	ControllerPort int
	AgentURL       string
	WSVersion      string
}

type AuthConf struct {
	AuthMethod string
	Keystone   auth.KeystoneParams
}

type LoggingConf struct {
	LogPath  string
	LogLevel string
}

type NetworkNameConf struct {
	VSwitchName  string
	VAdapterName string
}

type Configuration struct {
	Driver      DriverConf
	Auth        AuthConf
	Logging     LoggingConf
	NetworkName NetworkNameConf
}

func NewDefaultConfiguration() (conf Configuration) {
	conf.Driver.Adapter = "Ethernet0"
	conf.Driver.ControllerIP = "192.168.0.10"
	conf.Driver.ControllerPort = 8082
	conf.Driver.AgentURL = "http://127.0.0.1:9091"
	conf.Driver.WSVersion = "2016"

	conf.Logging.LogPath = logging.DefaultLogFilepath()
	conf.Logging.LogLevel = "Info"

	conf.Auth.AuthMethod = "noauth"

	conf.Auth.Keystone.Os_auth_url = ""
	conf.Auth.Keystone.Os_username = ""
	conf.Auth.Keystone.Os_tenant_name = ""
	conf.Auth.Keystone.Os_password = ""
	conf.Auth.Keystone.Os_token = ""

	return
}

func NewNetworkNameConfiguration(wsVersion string) (conf NetworkNameConf, err error) {
	err = nil
	switch wsVersion {
	case "2016":
		conf.VAdapterName = "vEthernet (HNSTransparent)"
		conf.VSwitchName = "Layered?<adapter>"
	case "2019":
		conf.VAdapterName = "vEthernet (<adapter>)"
		conf.VSwitchName = "ContrailRootNetwork*"
	default:
		err = errors.New("supported versions of Windows Server are: 2016, 2019")
	}
	return
}

func DefaultConfigFilepath() string {
	return string(filepath.Join(os.Getenv("ProgramData"),
		"Contrail", "etc", "contrail", "contrail-cnm-plugin.conf"))
}
