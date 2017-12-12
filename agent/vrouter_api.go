//
// Copyright (c) 2017 Juniper Networks, Inc. All Rights Reserved.
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

package agent

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/Juniper/contrail-windows-docker-driver/common"
)

func AddPort(vmUuid, vifUuid, ifName, mac, dockerID, ipAddress, vnUuid string) error {
	stdout, stderr, err := common.Call("python", common.AgentAPIWrapperScriptPath(),
		"add", vmUuid, vifUuid, fmt.Sprintf("\"%s\"", ifName), mac, dockerID,
		ipAddress, vnUuid)
	log.Debugf("Called Agent API wrapper: stdout: %s, stderr: %s", stdout, stderr)
	if err != nil {
		log.Errorf("When calling Agent API wrapper script: %s, %s", stdout, stderr)
		return err
	}
	return nil
}

func DeletePort(vifUuid string) error {
	stdout, stderr, err := common.Call("python", common.AgentAPIWrapperScriptPath(),
		"delete", vifUuid)
	log.Debugln("Called Agent API wrapper: ", stdout)
	if err != nil {
		log.Errorf("When calling Agent API wrapper script: %s, %s", stdout, stderr)
		return err
	}
	return nil
}
