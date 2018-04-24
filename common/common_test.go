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

package common_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/common/clock"

	"github.com/Juniper/contrail-windows-docker-driver/common/polling"

	"github.com/Juniper/contrail-windows-docker-driver/common"
	"github.com/Juniper/contrail-windows-docker-driver/common/networking"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Common suite")
}

type FakeInterface struct{ addrs []net.Addr }

func (iface *FakeInterface) Addrs() ([]net.Addr, error) { return iface.addrs, nil }

type FakeAddr struct{ addr string }

func (iface *FakeAddr) Network() string { return "foo" }
func (iface *FakeAddr) String() string  { return iface.addr }

var _ = Describe("WaitForInterface", func() {

	goodAddrs := []net.Addr{
		&FakeAddr{addr: "fe80::2498:43d5:1441:89b9/64"},
		&FakeAddr{addr: "172.16.0.3/16"},
	}

	ipv6Addrs := goodAddrs[:1]

	badAddrs := []net.Addr{
		&FakeAddr{addr: "fe80::2498:43d5:1441:89b9/64"},
		&FakeAddr{addr: "169.254.137.185/16"},
	}

	interfaceByName := func(ifname string) (networking.Interface, error) {
		switch ifname {
		case "good":
			return &FakeInterface{addrs: goodAddrs}, nil
		case "bad":
			return &FakeInterface{addrs: badAddrs}, nil
		case "ipv6":
			return &FakeInterface{addrs: ipv6Addrs}, nil
		default:
			return nil, fmt.Errorf("no such inteface")
		}
	}

	Specify("succeeds in simple case", func() {
		err := common.WaitForInterface(polling.NewOneShotPolicy(), interfaceByName, "good")
		Expect(err).ToNot(HaveOccurred())
	})

	Specify("fails on ipv6", func() {
		err := common.WaitForInterface(polling.NewOneShotPolicy(), interfaceByName, "ipv6")
		Expect(err).To(HaveOccurred())
	})

	Specify("fails on autoconf", func() {
		err := common.WaitForInterface(polling.NewOneShotPolicy(), interfaceByName, "bad")
		Expect(err).To(HaveOccurred())
	})

	Specify("fails on unknown interface", func() {
		err := common.WaitForInterface(polling.NewOneShotPolicy(), interfaceByName, "bad")
		Expect(err).To(HaveOccurred())
	})

	pollingPolicy := &polling.TimeoutPolicy{
		Timeout:         10 * time.Second,
		Delay:           1 * time.Second,
		Clock:           clock.NewFakeClock(),
		DelayMultiplier: 1,
	}

	Specify("fails with retry policy", func() {
		err := common.WaitForInterface(pollingPolicy, interfaceByName, "bad")
		Expect(err).To(HaveOccurred())
	})

	Specify("retries successfully", func() {
		attemptNo := 0

		var getter = func(string) (networking.Interface, error) {
			attemptNo += 1
			switch attemptNo {
			case 1:
				return nil, fmt.Errorf("interface not ready")
			case 2:
				return &FakeInterface{addrs: ipv6Addrs}, nil
			case 3:
				return &FakeInterface{addrs: badAddrs}, nil
			case 4:
				return &FakeInterface{addrs: goodAddrs}, nil
			default:
				return nil, fmt.Errorf("too many retries")
			}
		}

		err := common.WaitForInterface(pollingPolicy, getter, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(attemptNo).Should(Equal(4))
	})
})
