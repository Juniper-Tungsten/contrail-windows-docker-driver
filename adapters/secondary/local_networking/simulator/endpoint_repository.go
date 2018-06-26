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

	"github.com/Microsoft/hcsshim"
)

type InMemEndpointRepository struct{}

func (repo *InMemEndpointRepository) CreateEndpoint(configuration *hcsshim.HNSEndpoint) (string, error) {
	return "", errors.New("Not implemented yet")
}

func (repo *InMemEndpointRepository) GetEndpointByName(name string) (*hcsshim.HNSEndpoint, error) {
	return nil, errors.New("Not implemented yet")
}

func (repo *InMemEndpointRepository) DeleteEndpoint(endpointID string) error {
	return errors.New("Not implemented yet")
}
