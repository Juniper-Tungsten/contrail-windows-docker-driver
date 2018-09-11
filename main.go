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

package main

import (
	"errors"
	"flag"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/primary/cnm"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/api"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hns_contrail"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/vmswitch"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/port_association/agent"
	"github.com/Juniper/contrail-windows-docker-driver/configuration"
	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"
	"github.com/Juniper/contrail-windows-docker-driver/logging"
	log "github.com/sirupsen/logrus"
)

var (
	logPath = flag.String("logPath", logging.DefaultLogFilepath(),
		"Path to log file.")
	logLevelString = flag.String("logLevel", "Info",
		"Log verbosity (possible values: Debug|Info|Warn|Error|Fatal|Panic)")
	configPath = flag.String("config", configuration.DefaultConfigFilepath(),
		"Path to configuration file. See cnm-driver.conf.sample for an example.")
	dryRun = flag.Bool("dryRun", false,
		"Loads configuration but doesn't run anything. Useful for testing if config file syntax "+
			"is correct.")
)

func init() {
	flag.Parse()
}

func main() {
	logHook, err := logging.SetupHook(*logPath, *logLevelString)
	if err != nil {
		log.Errorf("Setting up logging failed: %s", err)
		os.Exit(1)
	}
	defer logHook.Close()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Errorln("Loading config failed:")
		os.Exit(2)
	}

	if *dryRun {
		log.Info("Dry run - exiting.")
		os.Exit(0)
	}

	err = run(cfg)
	if err != nil {
		log.Error(err)
		os.Exit(3)
	}
}

func loadConfig(cfgFilePath string) (*configuration.Configuration, error) {
	cfg := configuration.NewDefaultConfiguration()
	if cfgFilePath != "" {
		err := cfg.LoadFromFile(cfgFilePath)
		if err != nil {
			return nil, err
		}
	} else {
		cfg.LoadFromCommandLine()
	}

	cfg.Driver.VSwitchName = strings.Replace(cfg.Driver.VSwitchName, "<adapter>",
		cfg.Driver.Adapter, -1)

	log.Debugln("Configuration:", cfg)
	return &cfg, nil
}

func run(cfg *configuration.Configuration) error {
	if err := vmswitch.EnsureSwitchExists(cfg.Driver.VSwitchName, cfg.Driver.Adapter); err != nil {
		return err
	}
	hypervExtension := hyperv_extension.NewHyperVvRouterForwardingExtension(cfg.Driver.VSwitchName)
	vrouter := vrouter.NewHyperVvRouter(hypervExtension)

	controller, err := NewControllerAdapter(cfg)
	if err != nil {
		return err
	}

	agentUrl, err := url.Parse(cfg.Driver.AgentURL)
	if err != nil {
		return err
	}

	agent := agent.NewAgentRestAPI(http.DefaultClient, agentUrl)

	netRepo := hns_contrail.NewHNSContrailNetworksRepository(cfg.Driver.Adapter)

	epRepo := &hns_contrail.HNSEndpointRepository{}

	core, err := driver_core.NewContrailDriverCore(vrouter, controller, agent, netRepo, epRepo)
	if err != nil {
		return err
	}

	d := cnm.NewServerCNM(core)
	if err = d.StartServing(); err != nil {
		return err
	}
	defer d.StopServing()

	waitForSigInt()
	return nil
}

func waitForSigInt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Infoln("Good bye")
}

func NewControllerAdapter(cfg *configuration.Configuration) (
	*controller_rest.ControllerAdapter, error) {
	apiClient := api.NewApiClient(cfg.Driver.ControllerIP, cfg.Driver.ControllerPort)
	switch cfg.Auth.AuthMethod {
	case "keystone":
		return controller_rest.NewControllerWithKeystoneAdapter(cfg.Auth.Keystone, apiClient)
	case "noauth":
		return controller_rest.NewControllerInsecureAdapter(apiClient)
	default:
		return nil, errors.New("unsupported authentication method, use -authMethod flag")
	}
}
