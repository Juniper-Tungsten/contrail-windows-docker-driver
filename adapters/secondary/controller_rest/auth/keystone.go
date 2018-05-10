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

package auth

import (
	"errors"

	contrail "github.com/Juniper/contrail-go-api"
)

type KeystoneParams struct {
	Os_auth_url    string
	Os_username    string
	Os_tenant_name string
	Os_password    string
	Os_token       string
}

func NewKeystoneAuth(keys KeystoneParams) (*contrail.KeepaliveKeystoneClient, error) {
	if keys.Os_auth_url == "" {
		// this corner case is not handled by keystone.Authenticate. Causes panic.
		return nil, errors.New("Empty Keystone auth URL")
	}

	keystone := contrail.NewKeepaliveKeystoneClient(keys.Os_auth_url, keys.Os_tenant_name,
		keys.Os_username, keys.Os_password, keys.Os_token)
	return keystone, nil
}
