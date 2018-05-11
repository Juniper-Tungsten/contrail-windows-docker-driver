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
	"io/ioutil"

	"github.com/Juniper/contrail-windows-docker-driver/logging"
	"gopkg.in/gcfg.v1"
)

type Driver struct {
	Adapter             string
	ControllerIP        string
	ControllerPort      int
	AgentURL            string
	VSwitchNameWildcard string
	LogPath             string
	LogLevel            string
	ForceAsInteractive  bool
}

type Keystone struct {
	AuthUrl    string
	UserName   string
	TenantName string
	Password   string
	Token      string
}

type Configuration struct {
	Driver   Driver
	Keystone Keystone
}

func CreateDefaultConfiguration() (conf Configuration) {
	conf.Driver.Adapter = "Ethernet0"
	conf.Driver.ControllerIP = "127.0.0.1"
	conf.Driver.ControllerPort = 8082
	conf.Driver.AgentURL = "http://127.0.0.1:9091"
	conf.Driver.VSwitchNameWildcard = "Layered <adapter>"
	conf.Driver.LogPath = logging.DefaultLogFilepath()
	conf.Driver.LogLevel = "Info"
	conf.Driver.ForceAsInteractive = false

	conf.Keystone.AuthUrl = ""
	conf.Keystone.UserName = ""
	conf.Keystone.TenantName = ""
	conf.Keystone.Password = ""
	conf.Keystone.Token = ""

	return
}

func LoadConfigurationFromFile(filePath string) (Configuration, error) {
	configurationString, err := GetFileContentsAsString(filePath)
	if err != nil {
		return Configuration{}, err
	}
	return LoadConfigurationFromString(configurationString)
}

func GetFileContentsAsString(filePath string) (string, error) {
	configurationFileContents, err := ioutil.ReadFile(filePath)
	configurationString := string(configurationFileContents)
	return configurationString, err
}

func LoadConfigurationFromString(configurationString string) (Configuration, error) {
	conf := CreateDefaultConfiguration()
	err := gcfg.ReadStringInto(&conf, configurationString)
	return conf, err
}
