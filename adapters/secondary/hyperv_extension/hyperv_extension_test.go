// +build integration
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

package hyperv_extension_test

import (
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// NOTE: these tests require Administrator priveleges to run - hyperv_extension module will call
// {Get,...}-VMSwitch and {Get,...}-VMSwitchExtension commands and similar.

func TestHyperVExtension(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HyperVExtension")
}

var _ = Describe("HyperV Extension", func() {

	Context("Handling nonexisting vSwitch", func() {
		It("no switch with such name", func() {
			e := hyperv_extension.NewHyperVvRouterForwardingExtension("Nonexisting switch")
			_, err := e.IsEnabled()
			Expect(err).To(HaveOccurred())
		})

		It("vswitch name has wildcard, but it doesn't match anything anyways", func() {
			// In Powershell, if a name with wildcard doesn't yield a result, no error is thrown.
			// It's works like a naive filter. Therefore, if name with wildcard doesn't return
			// a result, we would like to know it as well.
			e := hyperv_extension.NewHyperVvRouterForwardingExtension("Nonexisting?switch")
			_, err := e.IsEnabled()
			Expect(err).To(HaveOccurred())
		})
	})
})
