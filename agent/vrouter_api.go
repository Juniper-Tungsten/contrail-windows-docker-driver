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
