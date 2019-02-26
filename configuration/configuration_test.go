// +build integration
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

//go:generate powershell ./New-BakedTestConfigFile.ps1

package configuration_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/auth"
	"github.com/Juniper/contrail-windows-docker-driver/configuration"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("configuration_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Configuration test suite",
		[]Reporter{junitReporter})
}

var _ = Describe("Configuration", func() {
	var cfg configuration.Configuration
	BeforeEach(func() {
		cfg = configuration.NewDefaultConfiguration()
	})

	Context("mapping and parsing the contents", func() {
		Context("when configuration string is empty", func() {
			It("error is not reported", func() {
				err := cfg.LoadFromString("")
				Expect(err).ToNot(HaveOccurred())
			})
			It("default configuration is loaded", func() {
				_ = cfg.LoadFromString("")
				defaultCfg := configuration.NewDefaultConfiguration()
				Expect(cfg).To(Equal(defaultCfg))
			})
		})
		Context("when configuration string is invalid", func() {
			It("error is reported", func() {
				err := cfg.LoadFromString("[DRIVER]\n\\")
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when some params or sections are not provided in configuration string", func() {
			It("error is not reported", func() {
				err := cfg.LoadFromString("[DRIVER]\nAdapter = Ethernet12\n")
				Expect(err).ToNot(HaveOccurred())
			})
			It("default values are used for unspecified parameters or sections", func() {
				cfg.LoadFromString("[DRIVER]\nAdapter = Ethernet12\n")
				defaultCfg := configuration.NewDefaultConfiguration()
				Expect(cfg.Driver.AgentURL).To(Equal(defaultCfg.Driver.AgentURL))
				Expect(cfg.Auth.Keystone).To(Equal(defaultCfg.Auth.Keystone))
			})
			It("specified parameters overwrite default ones", func() {
				// sanity check the default for the sake of the test.
				Expect(cfg.Driver.Adapter).To(Equal("Ethernet0"))

				cfg.LoadFromString("[DRIVER]\nAdapter = Ethernet12\n")
				Expect(cfg.Driver.Adapter).To(Equal("Ethernet12"))
			})
		})
	})

	Context("loading from file", func() {
		var confFile *os.File
		BeforeEach(func() {
			var err error
			confFile, err = ioutil.TempFile("", "test-")
			Expect(err).ToNot(HaveOccurred())
			// write contents baked into generated go file into the test conf file.
			_, err = confFile.Write([]byte(sampleCfgFile))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			confFile.Close()
			err := os.Remove(confFile.Name())
			Expect(err).ToNot(HaveOccurred())
		})

		It("loading reports error if file doesn't exist", func() {
			err := cfg.LoadFromFile("for_sure_there_is_no_such_a_file")
			Expect(err).To(HaveOccurred())
		})
		It("passes full example test case", func() {
			err := cfg.LoadFromFile(confFile.Name())
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Driver).To(Equal(configuration.DriverConf{
				Adapter:        "Ethernet0",
				ControllerIP:   "10.0.0.10",
				ControllerPort: 8082,
				AgentURL:       "http://127.0.0.1:9091",
				WSVersion:      "2016",
			}))
			Expect(cfg.Auth.AuthMethod).To(Equal("keystone"))
			Expect(cfg.Auth.Keystone).To(Equal(auth.KeystoneParams{
				Os_auth_url:    "http://10.0.0.10:5000/v2.0/",
				Os_username:    "admin",
				Os_tenant_name: "admin",
				Os_password:    "hunter2",
				Os_token:       "",
			}))
		})
	})
})
