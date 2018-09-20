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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	EmptyRequestResponseMessage = "Empty request/response."
	WrongHTTPMessageParameter   = "HTTPMessage run with wrong parameter (not request or response)"
	EmptyHTTPBody               = "Body is empty."
)

func SetupHook(logPath, logLevelString string) (*LogToFileHook, error) {
	logLevel, err := log.ParseLevel(logLevelString)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.SetLevel(logLevel)

	log.Debugln("Logging to", filepath.Dir(logPath))

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

// Function doesn't return error, because it is just for logging.
// If conversion to json returns error we want to log variable as raw
func VariableToJSON(variable interface{}) string {
	jsonOutput, err := json.Marshal(variable)
	if err != nil {
		log.Debugln("Converting to JSON error:", err)
		return fmt.Sprintf("Cannot convert request to JSON. Raw output: %s", variable)
	}
	return string(jsonOutput)
}

func readMessageBody(body io.ReadCloser) ([]byte, error) {
	var buf []byte
	var err error

	if body == nil {
		buf = []byte(EmptyHTTPBody)
	} else {
		buf, err = ioutil.ReadAll(body)
		if err != nil {
			log.Debugln("Cannot read request/response body.", err)
			buf = []byte("")
		}
	}
	return buf, err
}

func buildLogMessage(packageTag string, param interface{}, bodyBuffer *[]byte) string {

	var logMsg string

	switch param.(type) {
	case *http.Request:
		request := param.(*http.Request)
		// Body is a io.ReadCloser, so we need to construct one
		// in place of the one that we already read based on buffer
		// containing body content
		request.Body = ioutil.NopCloser(bytes.NewBuffer(*bodyBuffer))
		logMsg = fmt.Sprintf("[%s][%s]=>[%s] { Header: %s, Body: %s }", packageTag, request.Method, request.URL.String(), VariableToJSON(request.Header), *bodyBuffer)
	case *http.Response:
		response := param.(*http.Response)
		response.Body = ioutil.NopCloser(bytes.NewBuffer(*bodyBuffer))
		logMsg = fmt.Sprintf("[%s]{ Status: %s, Header: %s, Body: %s }", packageTag, response.Status, VariableToJSON(response.Header), *bodyBuffer)
	}

	return logMsg
}

func HTTPMessage(packageTag string, param interface{}) string {

	if param == nil {
		return EmptyRequestResponseMessage
	}

	var buf []byte

	switch paramType := param.(type) {
	default:
		log.Debugln(WrongHTTPMessageParameter)
		return WrongHTTPMessageParameter
	case *http.Request:
		if nil == paramType {
			return EmptyRequestResponseMessage
		} else {
			buf, _ = readMessageBody((param.(*http.Request)).Body)
		}
	case *http.Response:
		if nil == paramType {
			return EmptyRequestResponseMessage
		} else {
			buf, _ = readMessageBody((param.(*http.Response)).Body)
		}
	}

	return buildLogMessage(packageTag, param, &buf)
}
