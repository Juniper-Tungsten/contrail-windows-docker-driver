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

// Implemented according to
// https://github.com/docker/libnetwork/blob/master/docs/remote.md

package model

import (
	"net"
)

type Container struct {
	IP        net.IP
	PrefixLen int
	Mac       string
	Gateway   string
	VmUUID    string
	VmiUUID   string
	NetUUID   string
}

type Network struct {
	TenantName  string
	NetworkName string
	LocalID     string
	Subnet      Subnet
}

type LocalEndpoint struct {
	IfName string
	Name   string
}

type Subnet struct {
	DefaultGW     string
	CIDR          string
	DNSServerList []string
}
