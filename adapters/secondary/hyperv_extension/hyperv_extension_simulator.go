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

package hyperv_extension

type HyperVExtensionSimulator struct {
	Enabled bool
	Running bool
}

type HyperVExtensionFaultOnChangeSimulator struct {
	HyperVExtensionSimulator
	IsEnabledAfter bool
	IsRunningAfter bool
}

func (sim *HyperVExtensionSimulator) Enable() error {
	sim.Enabled = true
	return nil
}

func (sim *HyperVExtensionSimulator) Disable() error {
	sim.Running = false
	return nil
}

func (sim *HyperVExtensionSimulator) IsEnabled() (bool, error) {
	return sim.Enabled, nil
}

func (sim *HyperVExtensionSimulator) IsRunning() (bool, error) {
	return sim.Running, nil
}

func (faultsim *HyperVExtensionFaultOnChangeSimulator) Enable() error {
	faultsim.Enabled = faultsim.IsEnabledAfter
	faultsim.Running = faultsim.IsRunningAfter
	return nil
}

func (faultsim *HyperVExtensionFaultOnChangeSimulator) Disable() error {
	faultsim.Enabled = faultsim.IsEnabledAfter
	faultsim.Running = faultsim.IsRunningAfter
	return nil
}
