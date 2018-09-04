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

package helpers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Juniper/contrail-windows-docker-driver/powershell"
	log "github.com/sirupsen/logrus"
)

func HardResetHNS() error {
	log.Infoln("Resetting HNS")
	log.Debugln("Removing NAT")
	if _, _, err := powershell.CallPowershell("Get-NetNat", "|", "Remove-NetNat"); err != nil {
		log.Debugln("Could not remove nat network:", err)
	}
	log.Debugln("Removing container networks")
	if _, _, err := powershell.CallPowershell("Get-ContainerNetwork", "|", "Remove-ContainerNetwork",
		"-Force"); err != nil {
		log.Debugln("Could not remove container network:", err)
	}
	log.Debugln("Stopping HNS")
	if _, _, err := powershell.CallPowershell("Stop-Service", "hns"); err != nil {
		log.Debugln("HNS is already stopped:", err)
	}
	log.Debugln("Removing HNS program data")

	programData := os.Getenv("programdata")
	if programData == "" {
		return errors.New("Invalid program data env variable")
	}
	hnsDataDir := filepath.Join(programData, "Microsoft", "Windows", "HNS", "HNS.data")
	if _, _, err := powershell.CallPowershell("Remove-Item", hnsDataDir); err != nil {
		return fmt.Errorf("Error during removing HNS program data: %s", err)
	}
	log.Debugln("Starting HNS")
	if _, _, err := powershell.CallPowershell("Start-Service", "hns"); err != nil {
		return fmt.Errorf("Error when starting HNS: %s", err)
	}
	return nil
}

func RestartDocker() error {
	log.Infoln("Restarting docker")
	if _, _, err := powershell.CallPowershell("Restart-Service", "docker"); err != nil {
		return fmt.Errorf("When restarting docker: %s", err)
	}
	return nil
}
