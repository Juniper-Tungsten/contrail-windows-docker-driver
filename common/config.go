package common

import (
	"os"
	"path/filepath"

	"github.com/kardianos/osext"
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

	// AdapterReconnectTimeout is a time (in ms) to wait for adapter to reacquire IP after a new
	// HNS network is created. https://github.com/Microsoft/hcsshim/issues/108
	AdapterReconnectTimeout = 15000

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

	// AgentAPIWrapperScriptFileName is a file name of python script that calls vRouter Agent API
	AgentAPIWrapperScriptFileName = "agent_api.py"
)

// PluginSpecDir returns path to directory where docker daemon looks for plugin spec files.
func PluginSpecDir() string {
	return filepath.Join(os.Getenv("programdata"), "docker", "plugins")
}

// PluginSpecFilePath returns path to plugin spec file.
func PluginSpecFilePath() string {
	return filepath.Join(PluginSpecDir(), DriverName+".spec")
}

// AgentAPIWrapperScriptPath is path to python script that calls vRouter Agent API
func AgentAPIWrapperScriptPath() string {
	executable, _ := osext.Executable()
	return filepath.Join(filepath.Dir(executable), AgentAPIWrapperScriptFileName)
}
