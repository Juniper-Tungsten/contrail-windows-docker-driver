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

package simulator

import (
	"errors"

	"github.com/Juniper/contrail-windows-docker-driver/core/model"
	"github.com/Microsoft/hcsshim"
)

type InMemEndpointRepository struct {
	endpoints map[string]hcsshim.HNSEndpoint
}

func NewInMemEndpointRepository() *InMemEndpointRepository {
	return &InMemEndpointRepository{
		endpoints: make(map[string]hcsshim.HNSEndpoint),
	}
}

func (repo *InMemEndpointRepository) CreateEndpoint(name string, container *model.Container, network *model.Network) (string, error) {
	repo.endpoints[name] = hcsshim.HNSEndpoint{
		Id:   "123",
		Name: name,
	}
	return "123", nil
}

func (repo *InMemEndpointRepository) GetEndpoint(name string) (*hcsshim.HNSEndpoint, error) {
	if ep, exists := repo.endpoints[name]; exists {
		return &ep, nil
	} else {
		return nil, errors.New("endpoint not found")
	}
}

func (repo *InMemEndpointRepository) DeleteEndpoint(name string) error {
	if _, exists := repo.endpoints[name]; exists {
		delete(repo.endpoints, name)
		return nil
	} else {
		return errors.New("endpoint not found, so couldn't delete it")
	}
}
