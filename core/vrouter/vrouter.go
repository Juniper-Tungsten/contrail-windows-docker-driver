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

package vrouter

import (
	"errors"
	"fmt"
)

type VSwitchName string

type HyperVExtension interface {
	Enable() error
	Disable() error
	IsEnabled() (bool, error)
	IsRunning() (bool, error)
}

type HyperVvRouter struct {
	forwardingExtension HyperVExtension
}

func NewHyperVvRouter(fe HyperVExtension) *HyperVvRouter {
	return &HyperVvRouter{forwardingExtension: fe}
}

func (vr *HyperVvRouter) Initialize() error {
	if err := vr.assertRunning(); err != nil {
		return fmt.Errorf("Before trying to initialize vRouter: %s", err)
	}

	if enabled, err := vr.forwardingExtension.IsEnabled(); err != nil {
		return err
	} else if !enabled {
		if err := vr.forwardingExtension.Enable(); err != nil {
			return err
		}
	}

	if err := vr.assertEnabled(); err != nil {
		return fmt.Errorf("After trying to initialize vRouter: %s", err)
	}

	if err := vr.assertRunning(); err != nil {
		return fmt.Errorf("After trying to initialize vRouter: %s", err)
	}

	return nil
}

func (vr *HyperVvRouter) assertEnabled() error {
	if actuallyEnabled, err := vr.forwardingExtension.IsEnabled(); err != nil {
		return err
	} else if !actuallyEnabled {
		return errors.New("Extension is disabled, when it should be enabled")
	} else {
		return nil
	}
}

func (vr *HyperVvRouter) assertRunning() error {
	if actuallyRunning, err := vr.forwardingExtension.IsRunning(); err != nil {
		return err
	} else if !actuallyRunning {
		return errors.New("Extension is stopped, when it should be running - " +
			"possible fix involves reinstallation of the vRouter Forwarding Extension")
	} else {
		return nil
	}
}
