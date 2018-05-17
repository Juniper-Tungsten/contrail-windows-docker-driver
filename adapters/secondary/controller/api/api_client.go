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

package api

import (
	contrail "github.com/Juniper/contrail-go-api"
	"github.com/Juniper/contrail-go-api/mocks"
)

func NewFakeApiClient() contrail.ApiClient {
	mockedApiClient := new(mocks.ApiClient)
	mockedApiClient.Init()
	return mockedApiClient
}

func NewApiClient(ip string, port int, auth contrail.Authenticator) contrail.ApiClient {
	var realApiClient contrail.ApiClient
	realApiClient = contrail.NewClient(ip, port)
	realApiClient.(*contrail.Client).SetAuthenticator(auth)
	return realApiClient
}
