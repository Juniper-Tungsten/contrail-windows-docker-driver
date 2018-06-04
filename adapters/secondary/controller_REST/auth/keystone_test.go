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

package auth_test

import (
	"os"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestKeystone(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keystone")
}

var _ = Describe("Keystone", func() {
	Context("KeystoneParams", func() {
		type TestCase struct {
			keys         auth.KeystoneParams
			expectedKeys auth.KeystoneParams
		}
		DescribeTable("LoadFromEnvironment loads empty fields from env",
			func(t TestCase) {
				os.Setenv("OS_AUTH_URL", "http://1.3.3.7:5000/v2.0")
				os.Setenv("OS_USERNAME", "env_username")
				os.Setenv("OS_TENANT_NAME", "env_tenant")
				os.Setenv("OS_PASSWORD", "okon")
				t.keys.LoadFromEnvironment()
				Expect(t.keys).To(BeEquivalentTo(t.expectedKeys))
			},
			Entry("variables present", TestCase{
				keys: auth.KeystoneParams{
					Os_auth_url:    "http://1.2.3.4:5000/v2.0",
					Os_username:    "admin",
					Os_tenant_name: "admin",
					Os_password:    "hunter2",
					Os_token:       "",
				},
				expectedKeys: auth.KeystoneParams{
					Os_auth_url:    "http://1.2.3.4:5000/v2.0",
					Os_username:    "admin",
					Os_tenant_name: "admin",
					Os_password:    "hunter2",
					Os_token:       "",
				},
			}),
			Entry("variables empty (but are present in envs)", TestCase{
				keys: auth.KeystoneParams{
					Os_auth_url:    "",
					Os_username:    "",
					Os_tenant_name: "",
					Os_password:    "",
					Os_token:       "",
				},
				expectedKeys: auth.KeystoneParams{
					Os_auth_url:    "http://1.3.3.7:5000/v2.0",
					Os_username:    "env_username",
					Os_tenant_name: "env_tenant",
					Os_password:    "okon",
					Os_token:       "",
				},
			}),
		)
	})

	Context("NewKeystoneAuth", func() {
		type TestCase struct {
			shouldErr bool
			keys      auth.KeystoneParams
		}
		DescribeTable("NewKeystoneAuth",
			func(t TestCase) {
				_, err := auth.NewKeystoneAuth(&t.keys)
				if t.shouldErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("doesn't err if variables are not set except url", TestCase{
				keys: auth.KeystoneParams{
					Os_auth_url:    "http://1.2.3.4:5000/v2.0",
					Os_username:    "",
					Os_tenant_name: "",
					Os_password:    "",
					Os_token:       "",
				},
				shouldErr: false,
			}),
			Entry("errors if empty url", TestCase{
				keys: auth.KeystoneParams{
					Os_auth_url:    "",
					Os_username:    "admin",
					Os_tenant_name: "admin",
					Os_password:    "hunter2",
					Os_token:       "",
				},
				shouldErr: true,
			}),
			Entry("variables present", TestCase{
				keys: auth.KeystoneParams{
					Os_auth_url:    "http://1.2.3.4:5000/v2.0",
					Os_username:    "admin",
					Os_tenant_name: "admin",
					Os_password:    "hunter2",
					Os_token:       "",
				},
				shouldErr: false,
			}),
		)
	})
})
