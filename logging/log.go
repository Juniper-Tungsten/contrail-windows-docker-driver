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

package logging

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func SetupHook(logPath, logLevelString string) (*LogToFileHook, error) {
	logLevel, err := log.ParseLevel(logLevelString)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.SetLevel(logLevel)

	log.Infoln("Logging to", filepath.Dir(logPath))

	err = os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		return nil, fmt.Errorf("When trying to create log dir %s", err)
	}

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		return nil, fmt.Errorf("When trying to open log file: %s", err)
	}

	fileLoggerHook := NewLogToFileHook(logFile)
	log.AddHook(fileLoggerHook)

	return fileLoggerHook, nil
}

func DefaultLogFilepath() string {
	return string(filepath.Join(os.Getenv("ProgramData"),
		"Contrail", "var", "log", "contrail", "contrail-windows-docker-driver.log"))
}
