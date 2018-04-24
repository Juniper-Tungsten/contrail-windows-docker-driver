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

package clock

import "time"

type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
}

type RealClock struct{}

func (*RealClock) Now() time.Time                  { return time.Now() }
func (*RealClock) Since(t time.Time) time.Duration { return time.Since(t) }
func (*RealClock) Sleep(d time.Duration)           { time.Sleep(d) }

func NewRealClock() Clock {
	return &RealClock{}
}

type FakeClock struct {
	now time.Time
}

func (clock *FakeClock) Now() time.Time                  { return clock.now }
func (clock *FakeClock) Since(t time.Time) time.Duration { return clock.now.Sub(t) }
func (clock *FakeClock) Sleep(d time.Duration)           { clock.now = clock.now.Add(d) }

func NewFakeClock() *FakeClock {
	return &FakeClock{now: time.Now()}
}
