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

package main

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/agent"
	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/controller"
	"github.com/Juniper/contrail-windows-docker-driver/driver"
	"github.com/Juniper/contrail-windows-docker-driver/hnsManager"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

type WinService struct {
	adapter        string
	controllerIP   string
	controllerPort int
	vswitchName    string
	logDir         string
	logLevel       log.Level
	keys           controller.KeystoneEnvs
}

func main() {

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("Don't know if the session is interactive: %v", err)
	}

	var adapter = flag.String("adapter", "Ethernet0",
		"net adapter for HNS switch, must be physical")
	var controllerIP = flag.String("controllerIP", "127.0.0.1",
		"IP address of Contrail Controller API")
	var controllerPort = flag.Int("controllerPort", 8082,
		"port of Contrail Controller API")
	var logPath = flag.String("logPath", common.LogFilepath(), "log filepath")
	var logLevelString = flag.String("logLevel", "Info",
		"log verbosity (possible values: Debug|Info|Warn|Error|Fatal|Panic)")
	var vswitchNameWildcard = flag.String("vswitchName", "Layered <adapter>",
		"Name of Transparent virtual switch. Special wildcard \"<adapter>\" will be interpretted "+
			"as value of netAdapter parameter. For example, if netAdapter is \"Ethernet0\", then "+
			"vswitchName will equal \"Layered Ethernet0\". You can use Get-VMSwitch PowerShell "+
			"command to check how the switch is called on your version of OS.")
	var forceAsInteractive = flag.Bool("forceAsInteractive", false,
		"if true, will act as if ran from interactive mode. This is useful when running this "+
			"service from remote powershell session, because they're not interactive.")
	var os_auth_url = flag.String("os_auth_url", "", "Keystone auth url. If empty, will read "+
		"from environment variable")
	var os_username = flag.String("os_username", "", "Contrail username. If empty, "+
		"will read from environment variable")
	var os_tenant_name = flag.String("os_tenant_name", "", "Tenant name. If empty, will read "+
		"environment variable")
	var os_password = flag.String("os_password", "", "Contrail password. If empty, will read "+
		"environment variable")
	var os_token = flag.String("os_token", "", "Keystone token. If empty, will read "+
		"environment variable")
	flag.Parse()

	if *forceAsInteractive {
		isInteractive = true
	}

	vswitchName := strings.Replace(*vswitchNameWildcard, "<adapter>", *adapter, -1)

	logLevel, err := log.ParseLevel(*logLevelString)
	if err != nil {
		log.Error(err)
		return
	}
	log.SetLevel(logLevel)

	log.Infoln("Logging to", path.Dir(*logPath))

	err = os.MkdirAll(filepath.Dir(*logPath), 0755)
	if err != nil {
		log.Errorln("When trying to create log dir:", err)
	}

	logFile, err := os.OpenFile(*logPath, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		log.Errorln("When trying to open log file:", err)
	}
	defer logFile.Close()

	fileLoggerHook := common.NewLogToFileHook(logFile)
	log.AddHook(fileLoggerHook)

	keys := &controller.KeystoneEnvs{
		Os_auth_url:    *os_auth_url,
		Os_username:    *os_username,
		Os_tenant_name: *os_tenant_name,
		Os_password:    *os_password,
		Os_token:       *os_token,
	}
	keys.LoadFromEnvironment()

	winService := &WinService{
		adapter:        *adapter,
		controllerIP:   *controllerIP,
		controllerPort: *controllerPort,
		vswitchName:    vswitchName,
		logLevel:       logLevel,
		keys:           *keys,
	}

	svcRunFunc := debug.Run
	if !isInteractive {
		svcRunFunc = svc.Run
	}

	if err := svcRunFunc(common.WinServiceName, winService); err != nil {
		log.Errorf("%s service failed: %v", common.WinServiceName, err)
		return
	}
	log.Infof("%s service stopped", common.WinServiceName)
}

func (ws *WinService) Execute(args []string, winChangeReqChan <-chan svc.ChangeRequest,
	winStatusChan chan<- svc.Status) (ssec bool, errno uint32) {

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	winStatusChan <- svc.Status{State: svc.StartPending}

	c, err := controller.NewController(ws.controllerIP, ws.controllerPort, &ws.keys)
	if err != nil {
		log.Error(err)
		return
	}

	a, err := agent.NewAgent(agent.RestAPI)
	if err != nil {
		log.Error(err)
		return
	}

	d := driver.NewDriver(ws.adapter, ws.vswitchName, c, a, &hnsManager.HNSManager{})
	if err = d.StartServing(); err != nil {
		log.Error(err)
		return
	}
	defer d.StopServing()

	winStatusChan <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

win_svc_loop:
	for {
		svcCmd := <-winChangeReqChan

		switch svcCmd.Cmd {
		case svc.Interrogate:
			winStatusChan <- svcCmd.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			winStatusChan <- svcCmd.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break win_svc_loop
		default:
			log.Errorf("Unexpected control request #%d", svcCmd)
		}
	}
	winStatusChan <- svc.Status{State: svc.StopPending}
	return
}
