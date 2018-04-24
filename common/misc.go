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

package common

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/common/networking"
	"github.com/Juniper/contrail-windows-docker-driver/common/polling"
	log "github.com/sirupsen/logrus"
)

type VSwitchName string
type AdapterName string

func HardResetHNS() error {
	log.Infoln("Resetting HNS")
	log.Debugln("Removing NAT")
	if _, _, err := CallPowershell("Get-NetNat", "|", "Remove-NetNat"); err != nil {
		log.Debugln("Could not remove nat network:", err)
	}
	log.Debugln("Removing container networks")
	if _, _, err := CallPowershell("Get-ContainerNetwork", "|", "Remove-ContainerNetwork",
		"-Force"); err != nil {
		log.Debugln("Could not remove container network:", err)
	}
	log.Debugln("Stopping HNS")
	if _, _, err := CallPowershell("Stop-Service", "hns"); err != nil {
		log.Debugln("HNS is already stopped:", err)
	}
	log.Debugln("Removing HNS program data")

	programData := os.Getenv("programdata")
	if programData == "" {
		return errors.New("Invalid program data env variable")
	}
	hnsDataDir := filepath.Join(programData, "Microsoft", "Windows", "HNS", "HNS.data")
	if _, _, err := CallPowershell("Remove-Item", hnsDataDir); err != nil {
		return fmt.Errorf("Error during removing HNS program data: %s", err)
	}
	log.Debugln("Starting HNS")
	if _, _, err := CallPowershell("Start-Service", "hns"); err != nil {
		return fmt.Errorf("Error when starting HNS: %s", err)
	}
	return nil
}

func RestartDocker() error {
	log.Infoln("Restarting docker")
	if _, _, err := CallPowershell("Restart-Service", "docker"); err != nil {
		return fmt.Errorf("When restarting docker: %s", err)
	}
	return nil
}

func isAutoconfigurationIP(ip net.IP) bool {
	return ip[0] == 169 && ip[1] == 254
}

func WaitForInterface(policy polling.Policy, interfaceByName func(string) (networking.Interface, error), ifname AdapterName) error {
	sleeper := policy.Start()
	for {
		queryStart := time.Now()
		iface, err := interfaceByName(string(ifname))
		if err != nil {
			log.Warnf("Error when getting interface %s, but maybe it will appear soon: %s",
				ifname, err)
		} else {
			addrs, err := iface.Addrs()
			if err != nil {
				return err
			}

			// We print query time because it turns out that above operations actually take quite a
			// while (1-400ms), and the time depends (I think) on whether underlying interface
			// configs are being changed. For example, query usually takes ~10ms, but if it's about
			// to change, it can take up to 400ms. In other words, there must be some kind of mutex
			// there. This information could be useful for debugging.
			log.Debugf("Current %s addresses: %s. Query took %s", ifname,
				addrs, time.Since(queryStart))

			// We're essentialy waiting for adapter to reacquire IPv4 (that's how they do it
			// in Microsoft: https://github.com/Microsoft/hcsshim/issues/108)
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err == nil {
					ip = ip.To4()
					if ip != nil && !isAutoconfigurationIP(ip) {
						log.Debugf("Waited %s for IP reacquisition", sleeper.Elapsed())
						return nil
					}
				}
			}
		}

		if sleeper.Sleep() == polling.Stop {
			return errors.New("Waited for net adapter to reconnect for too long.")
		}
	}
}

func WaitForRealInterface(ifname AdapterName) error {
	return WaitForInterface(
		polling.NewTimeoutPolicy(AdapterReconnectTimeout, AdapterPollingRate),
		networking.InterfaceByName,
		ifname,
	)
}
