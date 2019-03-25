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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/Juniper/contrail-windows-docker-driver/logging"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
)

const (
	// ROOT_NETWORK_NAME is a name of root HNS network created solely for the purpose of
	// having a virtual switch
	ROOT_NETWORK_NAME = "ContrailRootNetwork"
)

type DriverConf struct {
	Adapter        string
	ControllerIP   string
	ControllerPort int
	AgentURL       string
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
	WSVersion   string
}

func NewDefaultConfiguration() (conf Configuration) {
	conf.Driver.Adapter = "Ethernet0"
	conf.Driver.ControllerIP = "192.168.0.10"
	conf.Driver.ControllerPort = 8082
	conf.Driver.AgentURL = "http://127.0.0.1:9091"

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

func GetWindowsServerVersion() (string, error) {
	cmd := exec.Command("reg.exe", "QUERY", "HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion", "/v", "Productname")
	outBuf, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	productName := string(outBuf)

	re := regexp.MustCompile(`Windows Server \d{4}`)
	version := re.FindString(productName)

	switch version {
	case "Windows Server 2016", "Windows Server 2019":
		return version[len(version)-4:], nil
	default:
		return "", errors.New("supported versions of Windows Server are: 2016, 2019")
	}
}

func NewNetworkNameConfiguration(cfg Configuration) (nameConf NetworkNameConf, err error) {
	switch cfg.WSVersion {
	case "2016":
		nameConf.VAdapterName = "vEthernet (HNSTransparent)"
		nameConf.VSwitchName = fmt.Sprintf("Layered?%s", cfg.Driver.Adapter)
	case "2019":
		nameConf.VAdapterName = fmt.Sprintf("vEthernet (%s)", cfg.Driver.Adapter)
		nameConf.VSwitchName = fmt.Sprintf("%s*", ROOT_NETWORK_NAME)
	}
	return
}

func DefaultConfigFilepath() string {
	return string(filepath.Join(os.Getenv("ProgramData"),
		"Contrail", "etc", "contrail", "contrail-cnm-plugin.conf"))
}
