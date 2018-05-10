//
// Copyright (c) 2017 Juniper Networks, Inc. All Rights Reserved.
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

package configuration

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("controller_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Configuration test suite",
		[]Reporter{junitReporter})
}

var _ = Describe("Configuration", func() {
	Describe("Loading configuration", func() {
		BeforeEach(func() {
		})
		Context("when configuration file doesn't exist", func() {
			It("loading reports error", func() {
				_, err := LoadConfigurationFromFile("for_sure_there_is_no_such_a_file")
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when configuration string is empty", func() {
			It("error is not reported", func() {
				_, err := LoadConfigurationFromString("")
				Expect(err).ToNot(HaveOccurred())
			})
			It("default configuration is loaded", func() {
				configuration, _ := LoadConfigurationFromString("")
				defaultConfiguration := CreateDefaultConfiguration()
				Expect(configuration).To(Equal(defaultConfiguration))
			})
		})
		Context("when configuration string is invalid", func() {
			It("error is reported", func() {
				_, err := LoadConfigurationFromString("[Driver]\n\\")
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when some parameters are not provided in configuration string", func() {
			It("error is not reported", func() {
				_, err := LoadConfigurationFromString("[Driver]\nadapter = Ethernet12\n")
				Expect(err).ToNot(HaveOccurred())
			})
			It("default values are used for unspecified parameters", func() {
				configuration, _ := LoadConfigurationFromString("[Driver]\nadapter = Ethernet12\n")
				defaultConfiguration := CreateDefaultConfiguration()
				Expect(configuration.Driver.AgentURL).To(Equal(defaultConfiguration.Driver.AgentURL))
			})
			It("specified parameters overwrite default ones", func() {
				adapter := "Ethernet12"
				configurationString := fmt.Sprintf("[Driver]\nadapter = %s\n", adapter)
				configuration, _ := LoadConfigurationFromString(configurationString)
				Expect(configuration.Driver.Adapter).To(Equal(adapter))
			})
		})
	})
})
