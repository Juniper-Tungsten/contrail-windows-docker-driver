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

package hns

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Microsoft/hcsshim"
	log "github.com/sirupsen/logrus"
)

const (
	// createHNSNetworkInitialRetryDelay is a delay between consecutive attemps
	// to create HNSNetwork. After each retry, the delay is increased to
	// give HNS more time to cool down.
	createHNSNetworkInitialRetryDelay = 5 * time.Second

	// createHNSNetworkTimeout is maximum waiting time for CreateHNSNetwork
	// in case of retrying. This should be greater than CreateHNSNetworkInitialRetryDelay
	// and AdapterReconnectTimeout combined for the retry to be attempted.
	createHNSNetworkTimeout = 90 * time.Second
)

type recoverableError struct {
	inner error
}

func (e *recoverableError) Error() string {
	return e.inner.Error()
}

func tryCreateHNSNetwork(config string) (string, error) {
	response, err := hcsshim.HNSNetworkRequest("POST", "", config)
	if err != nil {
		log.Errorln(err)

		errMsg := strings.ToLower(err.Error())
		// TODO: We need a comment explaining meaning behind these error messages and why it's ok
		// to ignore them for a bit.
		if strings.Contains(errMsg, "hns failed") && strings.Contains(errMsg, "unspecified error") {
			return "", &recoverableError{inner: err}
		} else {
			return "", err
		}
	}

	return response.Id, nil
}

func CreateHNSNetwork(configuration *hcsshim.HNSNetwork) (string, error) {
	log.Debugln("Creating HNS network")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config:", string(configBytes))

	var id = ""
	delay := createHNSNetworkInitialRetryDelay
	creatingStart := time.Now()
	for {
		id, err = tryCreateHNSNetwork(string(configBytes))
		if err != nil {
			if recoverableErr, ok := err.(*recoverableError); ok {
				err = recoverableErr.inner
				if time.Since(creatingStart) < createHNSNetworkTimeout {
					log.Warnln("Creating HNS network failed. Sleeping for ", delay, "ms before retrying.")
					time.Sleep(delay)
					delay *= 2
					continue
				}
			}
			return "", err
		}
		break
	}

	log.Infoln("Created HNS network with ID:", id)

	return id, nil
}

func DeleteHNSNetwork(hnsID string) error {
	log.Infoln("Deleting HNS network", hnsID)

	_, err := hcsshim.HNSNetworkRequest("DELETE", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}

	return nil
}

func ListHNSNetworks() ([]hcsshim.HNSNetwork, error) {
	log.Debugln("Listing HNS networks")
	nets, err := hcsshim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return nets, nil
}

func GetHNSNetwork(hnsID string) (*hcsshim.HNSNetwork, error) {
	log.Debugln("Getting HNS network", hnsID)
	net, err := hcsshim.HNSNetworkRequest("GET", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return net, nil
}

func GetHNSNetworkByName(name string) (*hcsshim.HNSNetwork, error) {
	log.Debugln("Getting HNS network by name:", name)
	nets, err := hcsshim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	for _, n := range nets {
		if n.Name == name {
			return &n, nil
		}
	}
	return nil, nil
}

func CreateHNSEndpoint(configuration *hcsshim.HNSEndpoint) (string, error) {
	log.Debugln("Creating HNS endpoint")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config: ", string(configBytes))
	response, err := hcsshim.HNSEndpointRequest("POST", "", string(configBytes))
	if err != nil {
		return "", err
	}
	log.Debugln("Created HNS endpoint with ID:", response.Id)
	return response.Id, nil
}

func DeleteHNSEndpoint(endpointID string) error {
	log.Debugln("Deleting HNS endpoint", endpointID)
	_, err := hcsshim.HNSEndpointRequest("DELETE", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func GetHNSEndpoint(endpointID string) (*hcsshim.HNSEndpoint, error) {
	log.Debugln("Getting HNS endpoint", endpointID)
	endpoint, err := hcsshim.HNSEndpointRequest("GET", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return endpoint, nil
}

func GetHNSEndpointByName(name string) (*hcsshim.HNSEndpoint, error) {
	log.Debugln("Getting HNS endpoint by name:", name)
	eps, err := hcsshim.HNSListEndpointRequest()
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	for _, ep := range eps {
		if ep.Name == name {
			return &ep, nil
		}
	}
	return nil, nil
}

func ListHNSEndpoints() ([]hcsshim.HNSEndpoint, error) {
	endpoints, err := hcsshim.HNSListEndpointRequest()
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

func ListHNSEndpointsOfNetwork(netID string) ([]hcsshim.HNSEndpoint, error) {
	eps, err := ListHNSEndpoints()
	if err != nil {
		return nil, err
	}
	var epsInNetwork []hcsshim.HNSEndpoint
	for _, ep := range eps {
		if ep.VirtualNetwork == netID {
			epsInNetwork = append(epsInNetwork, ep)
		}
	}
	return epsInNetwork, nil
}
