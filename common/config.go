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
	"os"
	"path/filepath"
)

const (
	// DomainName specifies domain name in Contrail
	DomainName = "default-domain"

	// DriverName is name of the driver that is to be specified during docker network creation
	DriverName = "Contrail"

	// HNSNetworkPrefix is a prefix given too all HNS network names managed by the driver
	HNSNetworkPrefix = "Contrail"

	// WinServiceName is the name of the Windows Service that the driver is ran as
	WinServiceName = "ContrailDockerDriver"

	// RootNetworkName is a name of root HNS network created solely for the purpose of
	// having a virtual switch
	RootNetworkName = "ContrailRootNetwork"

	// CreateHNSNetworkInitialRetryDelay is a delay (in ms) between consecutive attemps
	// to create HNSNetwork. After each retry, the delay is increased to
	// give HNS more time to cool down.
	CreateHNSNetworkInitialRetryDelay = 5 * 1000

	// CreateHNSNetworkTimeout is maximum waiting time (in ms) for CreateHNSNetwork
	// in case of retrying. This should be greater than CreateHNSNetworkInitialRetryDelay
	// and AdapterReconnectTimeout combined for the retry to be attempted.
	CreateHNSNetworkTimeout = 90 * 1000

	// AdapterReconnectMaxRetries is number of polling retries to wait for adapter
	// to reacquire IP after a new HNS network is created. https://github.com/Microsoft/hcsshim/issues/108
	AdapterReconnectMaxRetries = 30

	// AdapterPollingRate is rate (in ms) of polling of network adapter while waiting for it to
	// reacquire IP.
	AdapterPollingRate = 300

	// HNSTransparentInterfaceName is the name of transparent HNS vswitch interface name
	HNSTransparentInterfaceName = "vEthernet (HNSTransparent)"

	// PipePollingTimeout is time (in ms) to wait for named pipe to appear/disappear in the
	// filesystem
	PipePollingTimeout = 5000

	// PipePollingRate is rate (in ms) of polling named pipe if it appeared/disappeared in the
	// filesystem yet
	PipePollingRate = 300

	// HyperVExtensionName is the name of vRouter Hyper-V Extension
	HyperVExtensionName = "vRouter forwarding extension"
)

// PluginSpecDir returns path to directory where docker daemon looks for plugin spec files.
func PluginSpecDir() string {
	return filepath.Join(os.Getenv("programdata"), "docker", "plugins")
}

// PluginSpecFilePath returns path to plugin spec file.
func PluginSpecFilePath() string {
	return filepath.Join(PluginSpecDir(), DriverName+".spec")
}
