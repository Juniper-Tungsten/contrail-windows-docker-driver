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

package polling

import (
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/common/clock"
)

// Policy to be used for various polling loops
type Policy interface {
	Start() Sleeper
}

// Sleeper is one instance of polling policy executor
type Sleeper interface {
	// Checks if timeout was reached. If yes, returns false.
	// If no, sleeps and returns true.
	Sleep() Action

	// Returns a time elapsed (to be used only for debug purposes)
	Elapsed() time.Duration
}

type Action bool

const (
	// Retry the polling action once again
	Retry Action = true

	// Stop polling
	Stop Action = false
)

// Timeout --------------------------------------------------------------

type TimeoutPolicy struct {
	timeout         time.Duration
	delay           time.Duration
	delayMultiplier int
	clock           clock.Clock
}

func NewTimeoutPolicy(timeout, delay time.Duration) Policy {
	return NewExponentialBackoffPolicy(timeout, delay, 1)
}

func NewExponentialBackoffPolicy(timeout, delay time.Duration, delayMultiplier int) Policy {
	return &TimeoutPolicy{
		timeout:         timeout,
		delay:           delay,
		delayMultiplier: delayMultiplier,
		clock:           clock.NewRealClock(),
	}
}

func (policy *TimeoutPolicy) Start() Sleeper {
	return &timeoutSleeper{
		started: policy.clock.Now(),
		delay:   policy.delay,
		policy:  policy,
	}
}

type timeoutSleeper struct {
	started time.Time
	delay   time.Duration
	policy  *TimeoutPolicy
}

func (sleeper *timeoutSleeper) Sleep() Action {
	if sleeper.policy.clock.Since(sleeper.started) > sleeper.policy.timeout {
		return Stop
	} else {
		sleeper.policy.clock.Sleep(sleeper.delay)
		sleeper.delay *= time.Duration(sleeper.policy.delayMultiplier)
		return Retry
	}
}

func (sleeper *timeoutSleeper) Elapsed() time.Duration {
	return sleeper.policy.clock.Since(sleeper.started)
}

// One shot --------------------------------------------------------------

type oneShotPolicy struct{}

// NewOneShotPolicy creates a polling policy that never allows for retry
func NewOneShotPolicy() Policy        { return &oneShotPolicy{} }
func (*oneShotPolicy) Start() Sleeper { return &oneShotSleeper{} }

type oneShotSleeper struct{}

func (*oneShotSleeper) Sleep() Action          { return Stop }
func (*oneShotSleeper) Elapsed() time.Duration { return 0 }
