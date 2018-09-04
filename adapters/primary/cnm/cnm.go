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

// Implemented according to
// https://github.com/docker/libnetwork/blob/master/docs/remote.md

package cnm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/core/driver_core"
	winio "github.com/Microsoft/go-winio"
	"github.com/docker/go-connections/sockets"
	"github.com/docker/go-plugins-helpers/network"
	log "github.com/sirupsen/logrus"
)

const (
	// DriverName is name of the driver that is to be specified during docker network creation
	DriverName = "Contrail"

	// pipePollingTimeout is time to wait for named pipe to appear/disappear in the
	// filesystem
	pipePollingTimeout = 5 * time.Second

	// pipePollingRate is rate of polling named pipe if it appeared/disappeared in the
	// filesystem yet
	pipePollingRate = 300 * time.Millisecond
)

type ServerCNM struct {
	// TODO: for now, Core field is public, because we need to access its fields, like controller.
	// This should be made private when making the Controller port smaller.
	Core *driver_core.ContrailDriverCore
	// TODO: we need to keep the following fields for now, but the plan is to refactor them
	// out (along with related pipe logic) to a separate primary adapter.
	listener           net.Listener
	PipeAddr           string
	stopReasonChan     chan error
	stoppedServingChan chan interface{}
	IsServing          bool
}

type NetworkMeta struct {
	tenant     string
	network    string
	subnetCIDR string
}

func NewServerCNM(core *driver_core.ContrailDriverCore) *ServerCNM {
	d := &ServerCNM{
		Core:               core,
		PipeAddr:           "//./pipe/" + DriverName,
		stopReasonChan:     make(chan error, 1),
		stoppedServingChan: make(chan interface{}, 1),
		IsServing:          false,
	}
	return d
}

func (d *ServerCNM) StartServing() error {

	if d.IsServing {
		return errors.New("Already serving.")
	}

	startedServingChan := make(chan interface{}, 1)
	failedChan := make(chan error, 1)

	go func() {

		defer func() {
			d.IsServing = false
			d.stoppedServingChan <- true
		}()

		pipeConfig := winio.PipeConfig{
			// This will set permissions for Service, System, Adminstrator group and account to
			// have full access
			SecurityDescriptor: "D:(A;ID;FA;;;SY)(A;ID;FA;;;BA)(A;ID;FA;;;LA)(A;ID;FA;;;LS)",
			MessageMode:        true,
			InputBufferSize:    4096,
			OutputBufferSize:   4096,
		}

		var err error
		d.listener, err = winio.ListenPipe(d.PipeAddr, &pipeConfig)
		if err != nil {
			failedChan <- errors.New(fmt.Sprintln("When setting up listener:", err))
			return
		}

		if err := d.waitForPipeToAppear(); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When waiting for pipe to appear:", err))
			return
		}

		h := network.NewHandler(d)
		go func() {
			err := h.Serve(d.listener)
			if err != nil {
				d.stopReasonChan <- errors.New(fmt.Sprintln("When serving:", err))
			}
		}()

		if err := d.waitUntilPipeDialable(); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When waiting for pipe to be dialable:", err))
			return
		}

		if err := os.MkdirAll(d.pluginSpecDir(), 0755); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When setting up plugin spec directory:", err))
			return
		}

		url := "npipe://" + d.listener.Addr().String()
		if err := ioutil.WriteFile(d.PluginSpecFilePath(), []byte(url), 0644); err != nil {
			failedChan <- errors.New(fmt.Sprintln("When creating spec file:", err))
			return
		}

		d.IsServing = true
		startedServingChan <- true

		if err := <-d.stopReasonChan; err != nil {
			log.Errorln("Stopped serving because:", err)
		}

		log.Infoln("Closing npipe listener")
		if err := d.listener.Close(); err != nil {
			log.Warnln("When closing listener:", err)
		}

		log.Infoln("Removing spec file")
		if err := os.Remove(d.PluginSpecFilePath()); err != nil {
			log.Warnln("When removing spec file:", err)
		}

		if err := d.waitForPipeToStop(); err != nil {
			log.Warnln("Failed to properly close named pipe, but will continue anyways:", err)
		}
	}()

	select {
	case <-startedServingChan:
		log.Infoln("Started serving on ", d.PipeAddr)
		return nil
	case err := <-failedChan:
		log.Error(err)
		return err
	}
}

func (d *ServerCNM) StopServing() error {
	if d.IsServing {
		d.stopReasonChan <- nil
		<-d.stoppedServingChan
		log.Infoln("Stopped serving")
	}

	return nil
}

// PluginSpecFilePath returns path to plugin spec file.
func (d *ServerCNM) PluginSpecFilePath() string {
	return filepath.Join(d.pluginSpecDir(), DriverName+".spec")
}

func (d *ServerCNM) pluginSpecDir() string {
	// returns path to directory where docker daemon looks for plugin spec files.
	return filepath.Join(os.Getenv("programdata"), "docker", "plugins")
}

func (d *ServerCNM) waitForPipeToAppear() error {
	return d.waitForPipe(true)
}

func (d *ServerCNM) waitForPipeToStop() error {
	return d.waitForPipe(false)
}

func (d *ServerCNM) waitForPipe(waitUntilExists bool) error {
	timeStarted := time.Now()
	for {
		if time.Since(timeStarted) > pipePollingTimeout {
			return errors.New("Waited for pipe file for too long.")
		}

		_, err := os.Stat(d.PipeAddr)

		// if waitUntilExists is true, we wait for the file to appear in filesystem.
		// else, we wait for the file to disappear from the filesystem.
		if fileExists := !os.IsNotExist(err); fileExists == waitUntilExists {
			break
		} else {
			log.Warnf("Waiting for pipe file, but: %s", err)
		}

		time.Sleep(pipePollingRate)
	}

	return nil
}

func (d *ServerCNM) waitUntilPipeDialable() error {
	timeStarted := time.Now()
	for {
		if time.Since(timeStarted) > pipePollingTimeout {
			return errors.New("Waited for pipe to be dialable for too long.")
		}

		timeout := time.Millisecond * 10
		conn, err := sockets.DialPipe(d.PipeAddr, timeout)
		if err == nil {
			conn.Close()
			return nil
		}

		log.Warnf("Waiting until dialable, but: %s", err)

		time.Sleep(pipePollingRate)
	}
}
