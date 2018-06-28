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

package win_networking_test

import (
	"errors"
	"net"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns/win_networking"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestInterface(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interface")
}

type FakeInterface struct {
	addrs []net.Addr
}

func (iface *FakeInterface) Addrs() ([]net.Addr, error) { return iface.addrs, nil }

type FakeErroringInterface struct {
}

func (iface *FakeErroringInterface) Addrs() ([]net.Addr, error) { return nil, errors.New("abcd") }

type FakeAddr struct{ addr string }

func (iface *FakeAddr) Network() string { return "foo" }
func (iface *FakeAddr) String() string  { return iface.addr }

var _ = Describe("GetValidIpv4Address", func() {

	DescribeTable("GetValidIpv4Address non-erroring table",
		func(fakeAddrs []net.Addr, expectedValue net.IP, shouldError bool) {
			iface := &FakeInterface{
				addrs: fakeAddrs,
			}
			ip, err := win_networking.GetValidIpv4Address(iface)
			Expect(ip).To(Equal(expectedValue))
			if shouldError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("valid ipv4 address",
			[]net.Addr{
				&FakeAddr{addr: "1.2.3.4/24"},
			}, net.IPv4(1, 2, 3, 4).To4(), false),
		Entry("invalid ipv4 address",
			[]net.Addr{
				&FakeAddr{addr: "300.300.300.300/24"},
			}, nil, true),
		Entry("autoconf ipv4 address",
			[]net.Addr{
				&FakeAddr{addr: "169.254.0.1/16"},
			}, nil, true),
		Entry("valid ipv4 and ipv6 addresses",
			[]net.Addr{
				&FakeAddr{addr: "1.2.3.4/24"},
				&FakeAddr{addr: "fe80::2498:43d5:1441:89b9/64"},
			}, net.IPv4(1, 2, 3, 4).To4(), false),
	)

	It("returns nil and error if it doesn't have an ipv4 address", func() {
		iface := &FakeInterface{
			addrs: []net.Addr{
			// intentionally left empty
			},
		}
		ip, err := win_networking.GetValidIpv4Address(iface)
		Expect(ip).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("returns nil and error if couldn't retreive addresses", func() {
		iface := &FakeErroringInterface{}
		ip, err := win_networking.GetValidIpv4Address(iface)
		Expect(ip).To(BeNil())
		Expect(err).To(HaveOccurred())
	})
})
