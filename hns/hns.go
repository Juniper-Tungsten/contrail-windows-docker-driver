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

	"github.com/Juniper/contrail-windows-docker-driver/common"
	log "github.com/sirupsen/logrus"
)

type Hcs struct {
	shim HcsShim
}

func NewHcs(shim HcsShim) Hcs {
	return Hcs{shim: shim}
}

type recoverableError struct {
	inner error
}

func (e *recoverableError) Error() string {
	return e.inner.Error()
}

func (hcs Hcs) tryCreateHNSNetwork(config string) (string, error) {
	response, err := hcs.shim.HNSNetworkRequest("POST", "", config)
	if err != nil {
		log.Errorln(err)

		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "hns failed") && strings.Contains(errMsg, "unspecified error") {
			return "", &recoverableError{inner: err}
		}
		return "", err
	}

	// When the first HNS network is created, a vswitch is also created and attached to
	// specified network adapter. This adapter will temporarily lose network connectivity
	// while it reacquires IPv4. We need to wait for it.
	// https://github.com/Microsoft/hcsshim/issues/108
	if err := common.WaitForInterface(common.HNSTransparentInterfaceName); err != nil {
		log.Errorln(err)

		deleteErr := DeleteHNSNetwork(response.Id)
		if deleteErr != nil {
			return "", deleteErr
		}

		return "", &recoverableError{inner: err}
	}

	return response.Id, nil
}

func (hcs Hcs) CreateHNSNetwork(configuration *Network) (string, error) {
	log.Infoln("Creating HNS network")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config:", string(configBytes))

	var id = ""
	delay := time.Millisecond * common.CreateHNSNetworkInitialRetryDelay
	creatingStart := time.Now()
	for {
		id, err = hcs.tryCreateHNSNetwork(string(configBytes))
		if err != nil {
			if recoverableErr, ok := err.(*recoverableError); ok {
				err = recoverableErr.inner
				if time.Since(creatingStart) < time.Millisecond*common.CreateHNSNetworkTimeout {
					log.Infoln("Creating HNS network failed, retrying.")
					log.Infoln("Sleeping", delay, "ms")
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

func (hcs Hcs) DeleteHNSNetwork(hnsID string) error {
	log.Infoln("Deleting HNS network", hnsID)

	toDelete, err := GetHNSNetwork(hnsID)
	if err != nil {
		log.Errorln(err)
		return err
	}

	networks, err := ListHNSNetworks()
	if err != nil {
		log.Errorln(err)
		return err
	}

	adapterStillInUse := false
	for _, network := range networks {
		if network.Id != toDelete.Id &&
			network.NetworkAdapterName == toDelete.NetworkAdapterName {
			adapterStillInUse = true
			break
		}
	}

	_, err = hcs.shim.HNSNetworkRequest("DELETE", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}

	if !adapterStillInUse {
		// If the last network that uses an adapter is deleted, then the underlying vswitch is
		// also deleted. During this period, the adapter will temporarily lose network
		// connectivity while it reacquires IPv4. We need to wait for it.
		// https://github.com/Microsoft/hcsshim/issues/95
		if err := common.WaitForInterface(
			common.AdapterName(toDelete.NetworkAdapterName)); err != nil {
			log.Errorln(err)
			return err
		}
	}

	return nil
}

func (hcs Hcs) ListHNSNetworks() ([]Network, error) {
	log.Infoln("Listing HNS networks")
	nets, err := hcs.shim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return nets, nil
}

func (hcs Hcs) GetHNSNetwork(hnsID string) (*Network, error) {
	log.Infoln("Getting HNS network", hnsID)
	net, err := hcs.shim.HNSNetworkRequest("GET", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return net, nil
}

func (hcs Hcs) GetHNSNetworkByName(name string) (*Network, error) {
	log.Infoln("Getting HNS network by name:", name)
	nets, err := hcs.shim.HNSListNetworkRequest("GET", "", "")
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

func (hcs Hcs) CreateHNSEndpoint(configuration *Endpoint) (string, error) {
	log.Infoln("Creating HNS endpoint")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config: ", string(configBytes))
	response, err := hcs.shim.HNSEndpointRequest("POST", "", string(configBytes))
	if err != nil {
		return "", err
	}
	log.Infoln("Created HNS endpoint with ID:", response.Id)
	return response.Id, nil
}

func (hcs Hcs) DeleteHNSEndpoint(endpointID string) error {
	log.Infoln("Deleting HNS endpoint", endpointID)
	_, err := hcs.shim.HNSEndpointRequest("DELETE", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func (hcs Hcs) GetHNSEndpoint(endpointID string) (*Endpoint, error) {
	log.Infoln("Getting HNS endpoint", endpointID)
	endpoint, err := hcs.shim.HNSEndpointRequest("GET", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return endpoint, nil
}

func (hcs Hcs) GetHNSEndpointByName(name string) (*Endpoint, error) {
	log.Infoln("Getting HNS endpoint by name:", name)
	eps, err := hcs.shim.HNSListEndpointRequest()
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

func (hcs Hcs) ListHNSEndpoints() ([]Endpoint, error) {
	endpoints, err := hcs.shim.HNSListEndpointRequest()
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

func (hcs Hcs) ListHNSEndpointsOfNetwork(netID string) ([]Endpoint, error) {
	eps, err := ListHNSEndpoints()
	if err != nil {
		return nil, err
	}
	var epsInNetwork []Endpoint
	for _, ep := range eps {
		if ep.VirtualNetwork == netID {
			epsInNetwork = append(epsInNetwork, ep)
		}
	}
	return epsInNetwork, nil
}

// Legacy entrypoints:

func CreateHNSNetwork(configuration *Network) (string, error) {
	return NewHcs(&RealHcsShim{}).CreateHNSNetwork(configuration)
}

func DeleteHNSNetwork(hnsID string) error {
	return NewHcs(&RealHcsShim{}).DeleteHNSNetwork(hnsID)
}

func ListHNSNetworks() ([]Network, error) {
	return NewHcs(&RealHcsShim{}).ListHNSNetworks()
}

func GetHNSNetwork(hnsID string) (*Network, error) {
	return NewHcs(&RealHcsShim{}).GetHNSNetwork(hnsID)
}

func GetHNSNetworkByName(name string) (*Network, error) {
	return NewHcs(&RealHcsShim{}).GetHNSNetworkByName(name)
}

func CreateHNSEndpoint(configuration *Endpoint) (string, error) {
	return NewHcs(&RealHcsShim{}).CreateHNSEndpoint(configuration)
}

func DeleteHNSEndpoint(endpointID string) error {
	return NewHcs(&RealHcsShim{}).DeleteHNSEndpoint(endpointID)
}

func GetHNSEndpoint(endpointID string) (*Endpoint, error) {
	return NewHcs(&RealHcsShim{}).GetHNSEndpoint(endpointID)
}

func GetHNSEndpointByName(name string) (*Endpoint, error) {
	return NewHcs(&RealHcsShim{}).GetHNSEndpointByName(name)
}

func ListHNSEndpoints() ([]Endpoint, error) {
	return NewHcs(&RealHcsShim{}).ListHNSEndpoints()
}

func ListHNSEndpointsOfNetwork(netID string) ([]Endpoint, error) {
	return NewHcs(&RealHcsShim{}).ListHNSEndpointsOfNetwork(netID)
}
