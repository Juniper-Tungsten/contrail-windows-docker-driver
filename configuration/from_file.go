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
	"github.com/go-ini/ini"
)

// Method used for simpler testing
func (conf *Configuration) LoadFromString(data string) error {
	iniCfg, err := ini.Load([]byte(data))
	if err != nil {
		return err
	}

	return conf.mapIni(iniCfg)
}

func (conf *Configuration) LoadFromFile(filepath string) error {
	iniCfg, err := ini.Load(filepath)
	if err != nil {
		return err
	}

	return conf.mapIni(iniCfg)
}

func (conf *Configuration) mapIni(cfg *ini.File) error {
	err := cfg.Section("DRIVER").MapTo(&conf.Driver)
	if err != nil {
		return err
	}

	err = cfg.Section("LOGGING").MapTo(&conf.Logging)
	if err != nil {
		return err
	}

	err = cfg.Section("AUTH").MapTo(&conf.Auth)
	if err != nil {
		return err
	}

	err = cfg.Section("KEYSTONE").MapTo(&conf.Auth.Keystone)
	if err != nil {
		return err
	}

	return nil
}
