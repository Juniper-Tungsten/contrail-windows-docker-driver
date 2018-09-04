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

package win_networking

import (
	"errors"
	"net"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns/win_networking/retry"
)

const (
	// adapterReconnectMaxRetries is number of polling retries to wait for adapter
	// to reacquire IP after a new HNS network is created. https://github.com/Microsoft/hcsshim/issues/108
	adapterReconnectMaxRetries = 30

	// adapterPollingRate is rate of polling of network adapter while waiting for it to
	// reacquire IP.
	adapterPollingRate = 300 * time.Millisecond
)

type Interface interface {
	Addrs() ([]net.Addr, error)
}

func WaitForValidIPReacquisition(ifname string) error {
	iface, err := pollForInterfaceToAppearInOS(ifname)
	if err != nil {
		return err
	}
	return waitForValidIPv4Address(iface)
}

func pollForInterfaceToAppearInOS(ifname string) (*net.Interface, error) {
	var iface *net.Interface = nil
	ifaceGetterFunc := func() error {
		var err error
		iface, err = net.InterfaceByName(ifname)
		return err
	}
	err := retry.Retry(ifaceGetterFunc, adapterReconnectMaxRetries, adapterPollingRate)
	return iface, err
}

func waitForValidIPv4Address(iface Interface) error {
	ifaceCheckerFunc := func() error {
		_, err := GetValidIpv4Address(iface)
		return err
	}
	return retry.Retry(ifaceCheckerFunc, adapterReconnectMaxRetries, adapterPollingRate)
}

func GetValidIpv4Address(iface Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err == nil {
			ip = ip.To4()
			if ip != nil && !isAutoconfigurationIP(ip) {
				return ip, nil
			}
		}
	}
	return nil, errors.New("No valid IPv4 address found")
}

func isAutoconfigurationIP(ip net.IP) bool {
	return ip[0] == 169 && ip[1] == 254
}
