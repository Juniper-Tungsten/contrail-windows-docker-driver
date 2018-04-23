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
	"github.com/Juniper/contrail-windows-docker-driver/common/nal"
	log "github.com/sirupsen/logrus"
)

type Hns struct {
	shim HcsShim
	nal  nal.Nal
}

func RealHns() Hns {
	return Hns{
		shim: &RealHcsShim{},
		nal:  nal.RealNal(),
	}
}

type recoverableError struct {
	inner error
}

func (e *recoverableError) Error() string {
	return e.inner.Error()
}

func (hns Hns) tryCreateHNSNetwork(config string) (string, error) {
	response, err := hns.shim.HNSNetworkRequest("POST", "", config)
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
	if err := hns.nal.WaitForInterface(common.HNSTransparentInterfaceName); err != nil {
		log.Errorln(err)

		deleteErr := DeleteHNSNetwork(response.Id)
		if deleteErr != nil {
			return "", deleteErr
		}

		return "", &recoverableError{inner: err}
	}

	return response.Id, nil
}

func (hns Hns) CreateHNSNetwork(configuration *Network) (string, error) {
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
		id, err = hns.tryCreateHNSNetwork(string(configBytes))
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

func (hns Hns) DeleteHNSNetwork(hnsID string) error {
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

	_, err = hns.shim.HNSNetworkRequest("DELETE", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}

	if !adapterStillInUse {
		// If the last network that uses an adapter is deleted, then the underlying vswitch is
		// also deleted. During this period, the adapter will temporarily lose network
		// connectivity while it reacquires IPv4. We need to wait for it.
		// https://github.com/Microsoft/hcsshim/issues/95
		if err := hns.nal.WaitForInterface(
			common.AdapterName(toDelete.NetworkAdapterName)); err != nil {
			log.Errorln(err)
			return err
		}
	}

	return nil
}

func (hns Hns) ListHNSNetworks() ([]Network, error) {
	log.Infoln("Listing HNS networks")
	nets, err := hns.shim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return nets, nil
}

func (hns Hns) GetHNSNetwork(hnsID string) (*Network, error) {
	log.Infoln("Getting HNS network", hnsID)
	net, err := hns.shim.HNSNetworkRequest("GET", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return net, nil
}

func (hns Hns) GetHNSNetworkByName(name string) (*Network, error) {
	log.Infoln("Getting HNS network by name:", name)
	nets, err := hns.shim.HNSListNetworkRequest("GET", "", "")
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

func (hns Hns) CreateHNSEndpoint(configuration *Endpoint) (string, error) {
	log.Infoln("Creating HNS endpoint")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config: ", string(configBytes))
	response, err := hns.shim.HNSEndpointRequest("POST", "", string(configBytes))
	if err != nil {
		return "", err
	}
	log.Infoln("Created HNS endpoint with ID:", response.Id)
	return response.Id, nil
}

func (hns Hns) DeleteHNSEndpoint(endpointID string) error {
	log.Infoln("Deleting HNS endpoint", endpointID)
	_, err := hns.shim.HNSEndpointRequest("DELETE", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func (hns Hns) GetHNSEndpoint(endpointID string) (*Endpoint, error) {
	log.Infoln("Getting HNS endpoint", endpointID)
	endpoint, err := hns.shim.HNSEndpointRequest("GET", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return endpoint, nil
}

func (hns Hns) GetHNSEndpointByName(name string) (*Endpoint, error) {
	log.Infoln("Getting HNS endpoint by name:", name)
	eps, err := hns.shim.HNSListEndpointRequest()
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

func (hns Hns) ListHNSEndpoints() ([]Endpoint, error) {
	endpoints, err := hns.shim.HNSListEndpointRequest()
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

func (hns Hns) ListHNSEndpointsOfNetwork(netID string) ([]Endpoint, error) {
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
	return RealHns().CreateHNSNetwork(configuration)
}

func DeleteHNSNetwork(hnsID string) error {
	return RealHns().DeleteHNSNetwork(hnsID)
}

func ListHNSNetworks() ([]Network, error) {
	return RealHns().ListHNSNetworks()
}

func GetHNSNetwork(hnsID string) (*Network, error) {
	return RealHns().GetHNSNetwork(hnsID)
}

func GetHNSNetworkByName(name string) (*Network, error) {
	return RealHns().GetHNSNetworkByName(name)
}

func CreateHNSEndpoint(configuration *Endpoint) (string, error) {
	return RealHns().CreateHNSEndpoint(configuration)
}

func DeleteHNSEndpoint(endpointID string) error {
	return RealHns().DeleteHNSEndpoint(endpointID)
}

func GetHNSEndpoint(endpointID string) (*Endpoint, error) {
	return RealHns().GetHNSEndpoint(endpointID)
}

func GetHNSEndpointByName(name string) (*Endpoint, error) {
	return RealHns().GetHNSEndpointByName(name)
}

func ListHNSEndpoints() ([]Endpoint, error) {
	return RealHns().ListHNSEndpoints()
}

func ListHNSEndpointsOfNetwork(netID string) ([]Endpoint, error) {
	return RealHns().ListHNSEndpointsOfNetwork(netID)
}
