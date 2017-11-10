package hyperv

import (
	"errors"
	"fmt"

	"github.com/Juniper/contrail-windows-docker-driver/common"
	log "github.com/sirupsen/logrus"
)

func EnableExtension(vswitchName common.VSwitchName) error {
	log.Infoln("Enabling vRouter Hyper-V Extension")
	if out, err := callOnSwitch(vswitchName, "Enable-VMSwitchExtension"); err != nil {
		log.Errorf("When enabling Hyper-V Extension: %s, %s", err, out)
		return err
	}

	enabled, err := IsExtensionEnabled(vswitchName)
	if err != nil {
		return err
	}

	if !enabled {
		return errors.New("extension is not enabled after being enabled")
	}

	return nil
}

func DisableExtension(vswitchName common.VSwitchName) error {
	log.Infoln("Disabling vRouter Hyper-V Extension")
	if out, err := callOnSwitch(vswitchName, "Disable-VMSwitchExtension"); err != nil {
		log.Errorf("When disabling Hyper-V Extension: %s, %s", err, out)
		return err
	}
	return nil
}

func IsExtensionEnabled(vswitchName common.VSwitchName) (bool,
	error) {
	out, err := inspectExtensionProperty(vswitchName, "Enabled")
	if err != nil {
		log.Errorf("When inspecting Hyper-V Extension: %s, %s", err, out)
		return false, err
	}
	return out == "True", nil
}

func IsExtensionRunning(vswitchName common.VSwitchName) (bool,
	error) {
	out, err := inspectExtensionProperty(vswitchName, "Running")
	if err != nil {
		log.Errorf("When inspecting Hyper-V Extension: %s, %s", err, out)
		return false, err
	}
	return out == "True", nil
}

func inspectExtensionProperty(vswitchName common.VSwitchName, property string) (string, error) {
	log.Infoln("Inspecting vRouter Hyper-V Extension for property:", property)
	// we use -Expand, because otherwise, we get an object instead of single string value
	out, err := callOnSwitch(vswitchName, "Get-VMSwitchExtension", "|", "Select",
		"-Expand", fmt.Sprintf("\"%s\"", property))
	log.Debugln("Inspect result:", out)
	return out, err
}

func callOnSwitch(vswitchName common.VSwitchName, command string, optionals ...string) (string,
	error) {
	c := []string{command,
		"-VMSwitchName", fmt.Sprintf("\"%s\"", string(vswitchName)),
		"-Name", fmt.Sprintf("\"%s\"", common.HyperVExtensionName)}
	for _, opt := range optionals {
		c = append(c, opt)
	}
	stdout, _, err := common.CallPowershell(c...)
	return stdout, err
}
