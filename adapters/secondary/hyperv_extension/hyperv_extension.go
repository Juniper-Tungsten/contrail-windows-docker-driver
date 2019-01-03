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

package hyperv_extension

import (
	"errors"
	"fmt"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/vmswitch"
	"github.com/Juniper/contrail-windows-docker-driver/powershell"
	log "github.com/sirupsen/logrus"
)

type hyperVvRouterForwardingExtension struct {
	vswitchName   string
	extensionName string
}

const hyperVvRouterForwardingExtensionName = "vRouter forwarding extension"

func NewHyperVvRouterForwardingExtension(vswitchName string) *hyperVvRouterForwardingExtension {
	return &hyperVvRouterForwardingExtension{
		vswitchName:   vswitchName,
		extensionName: hyperVvRouterForwardingExtensionName,
	}
}

func (hvvr *hyperVvRouterForwardingExtension) Enable() error {
	log.Infoln("Enabling Hyper-V Extension")
	if out, err := hvvr.callOnSwitchExtension("Enable-VMSwitchExtension"); err != nil {
		log.Errorf("When enabling Hyper-V Extension: %s, %s", err, out)
		return err
	}
	return nil
}

func (hvvr *hyperVvRouterForwardingExtension) Disable() error {
	log.Infoln("Disabling Hyper-V Extension")
	if out, err := hvvr.callOnSwitchExtension("Disable-VMSwitchExtension"); err != nil {
		log.Errorf("When disabling Hyper-V Extension: %s, %s", err, out)
		return err
	}
	return nil
}

func (hvvr *hyperVvRouterForwardingExtension) IsEnabled() (bool, error) {
	out, err := hvvr.inspectExtensionProperty("Enabled")
	if err != nil {
		log.Errorf("When inspecting Hyper-V Extension: %s, %s", err, out)
		return false, err
	}
	return out == "True", nil
}

func (hvvr *hyperVvRouterForwardingExtension) IsRunning() (bool, error) {
	out, err := hvvr.inspectExtensionProperty("Running")
	if err != nil {
		log.Errorf("When inspecting Hyper-V Extension: %s, %s", err, out)
		return false, err
	}
	return out == "True", nil
}

func (hvvr *hyperVvRouterForwardingExtension) inspectExtensionProperty(property string) (string, error) {
	log.Debugln("Inspecting Hyper-V Extension for property:", property)
	// we use -Expand, because otherwise, we get an object instead of single string value
	out, err := hvvr.callOnSwitchExtension("Get-VMSwitchExtension", "|", "Select",
		"-Expand", fmt.Sprintf("\"%s\"", property))
	log.Debugln("Inspect result:", out)
	return out, err
}

func (hvvr *hyperVvRouterForwardingExtension) callOnSwitchExtension(command string, optionals ...string) (string,
	error) {

	if switchState, err := vmswitch.DoesSwitchExist(hvvr.vswitchName); err != nil {
		return "", err
	} else if switchState == vmswitch.DELETED || switchState == vmswitch.DELETING {
		return "", errors.New("could not perform action on vmswitch extension, because vmswitch was not found")
	}

	c := []string{command,
		"-VMSwitchName", fmt.Sprintf("\"%s\"", hvvr.vswitchName),
		"-Name", fmt.Sprintf("\"%s\"", hvvr.extensionName)}
	for _, opt := range optionals {
		c = append(c, opt)
	}
	stdout, _, err := powershell.CallPowershell(c...)
	return stdout, err
}
