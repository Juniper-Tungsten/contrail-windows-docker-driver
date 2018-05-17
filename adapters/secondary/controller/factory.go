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

package controller

import (
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller/api"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller/auth"
)

func NewControllerWithKeystone(keys *auth.KeystoneParams, ip string, port int) (*Controller, error) {
	auth, err := auth.NewKeystoneAuth(keys)
	if err != nil {
		return nil, err
	}

	err = auth.Authenticate()
	if err != nil {
		return nil, err
	}

	apiClient := api.NewApiClient(ip, port, auth)

	c, err := newController(apiClient)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func NewFakeController() *Controller {
	fakeApiClient := api.NewFakeApiClient()
	c, _ := newController(fakeApiClient) // can't fail
	return c
}
