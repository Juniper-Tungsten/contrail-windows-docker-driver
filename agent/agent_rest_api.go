//
// Copyright (c) 2017 Juniper Networks, Inc. All Rights Reserved.
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

package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const {
	defaultAgentUrl = "127.0.0.1:1234"
}

type portRequestMsg struct {
	Time        string `json:"time"`
	VmUUID      string `json:"instance-id"`
	VifUUID     string `json:"id"`
	IfName      string `json:"system-name"`
	Mac         string `json:"mac-address"`
	DockerID    string `json:"display-name"`
	IpAddress   string `json:"ip-address"`
	VnUUID      string `json:"vn-id"`
	Ipv6        string `json:"ip6-address"`
	Type        int    `json:"type"`
	RxVlanId    int    `json:"rx-vlan-id"`
	TxVlanId    int    `json:"tx-vlan-id"`
	VmProjectId string `json:"vm-project-id"`
}

type agentRestAPI struct {
	httpClient *http.Client
	agentUrl string
}

func NewAgentRestAPI(httpClient *http.Client) *agentRestAPI {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &agentRestAPI{httpClient, defaultAgentUrl}
}

func (agent *agentRestAPI) AddPort(vmUUID, vifUUID, ifName, mac, dockerID, ipAddress, vnUUID string) error {
	t := time.Now()
	msg := portRequestMsg {
		Time:        t.String(),
		VmUUID:      vmUUID,
		VifUUID:     vifUUID,
		IfName:      ifName,
		Mac:         mac,
		DockerID:    dockerID,
		IpAddress:   ipAddress,
		VnUUID:      vnUUID,
		Ipv6:        "",
		Type:        1,
		RxVlanId:    -1,
		TxVlanId:    -1,
		VmProjectId: ""
	}

	msgBody := json.MarshalIndent(msg, "", "\t")
	fmt.Println(msgBody)

	response, error := agent.httpClient.Post(agentUrl + "/port", msgBody, nil)
	if error != nil {
		return error
	}

	if response.StatusCode == StatusOK {
		response.Body.Close()
		return nil
	} else {
		return fmt.Errorf(
			"Agent rest API: add port request failed (port = %s; status code = %d)",
			vifUUID, response.StatusCode)
	}
}

func (agent *agentRestAPI) DeletePort(vifUUID string) error {
	response, error := agent.httpClient.Do(http.NewRequest("DELETE",
		agentUrl + "/port/" + vifUUID, nil))
	if error != nil {
		return error
	}

	if response.StatusCode == StatusOK {
		response.Body.Close()
		return nil
	} else {
		return fmt.Errorf(
			"Agent rest API: delete port request failed (port = %s; status code = %d)",
			vifUUID, response.StatusCode)
	}
}
