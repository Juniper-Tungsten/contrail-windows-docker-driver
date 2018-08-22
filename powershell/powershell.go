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

package powershell

import (
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Call(command string, args ...string) (string, string, error) {
	cmd := exec.Command(command, args...)

	log.Debugf("Running %s: %s. ", command, strings.Join(args, " "))

	stdoutPipe, stderrPipe, err := setupOutputCollection(cmd)
	if err != nil {
		return "", "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", "", err
	}

	stdout, stderr, err := collectOutput(stdoutPipe, stderrPipe)
	if err != nil {
		return "", "", err
	}
	stdout = strings.TrimSpace(stdout)

	printDebugInfo(stdout, stderr)

	err = cmd.Wait()

	return stdout, stderr, err
}

func CallPowershell(args ...string) (string, string, error) {
	c := []string{"-NonInteractive"}
	for _, arg := range args {
		c = append(c, arg)
	}

	return Call("powershell", c...)
}

func setupOutputCollection(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}
	return stdoutPipe, stderrPipe, err
}

func collectOutput(stdoutPipe, stderrPipe io.ReadCloser) (string, string, error) {
	outBuf, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		return "", "", err
	}
	errBuf, err := ioutil.ReadAll(stderrPipe)
	if err != nil {
		return "", "", err
	}

	// ReadAll returns in []byte, so convert to string
	return fmt.Sprintf("%s", outBuf), fmt.Sprintf("%s", errBuf), nil
}

func printDebugInfo(stdout, stderr string) {
	logMsg := ""
	if stdout != "" {
		logMsg += fmt.Sprintf("stdout: %s;", stdout)
	}
	if stderr != "" {
		logMsg += fmt.Sprintf("stderr: %s;", stderr)
	}
	if logMsg != "" {
		log.Debug(logMsg)
	}
}
