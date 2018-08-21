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
	"os"
	"reflect"

	contrail "github.com/Juniper/contrail-go-api"
	log "github.com/sirupsen/logrus"
)

type KeystoneParams struct {
	Os_auth_url    string
	Os_username    string
	Os_tenant_name string
	Os_password    string
	Os_token       string
}

func NewKeystoneAuth(keys *KeystoneParams) (*contrail.KeepaliveKeystoneClient, error) {
	if keys.Os_auth_url == "" {
		// this corner case is not handled by keystone.Authenticate. Causes panic.
		return nil, errors.New("Empty Keystone auth URL")
	}

	keystone := contrail.NewKeepaliveKeystoneClient(keys.Os_auth_url, keys.Os_tenant_name,
		keys.Os_username, keys.Os_password, keys.Os_token)
	return keystone, nil
}

func (k *KeystoneParams) LoadFromEnvironment() {

	k.Os_auth_url = getenvIfNil(k.Os_auth_url, "OS_AUTH_URL")
	k.Os_username = getenvIfNil(k.Os_username, "OS_USERNAME")
	k.Os_tenant_name = getenvIfNil(k.Os_tenant_name, "OS_TENANT_NAME")
	k.Os_password = getenvIfNil(k.Os_password, "OS_PASSWORD")
	k.Os_token = getenvIfNil(k.Os_token, "OS_TOKEN")

	// print a warning for every empty variable
	keysReflection := reflect.ValueOf(*k)
	for i := 0; i < keysReflection.NumField(); i++ {
		if keysReflection.Field(i).String() == "" {
			log.Warn("Keystone variable empty: ", keysReflection.Type().Field(i).Name)
		}
	}
	log.Debugln(k)
}

func getenvIfNil(currentVal, envVar string) string {
	if currentVal == "" {
		return os.Getenv(envVar)
	}
	return currentVal
}
