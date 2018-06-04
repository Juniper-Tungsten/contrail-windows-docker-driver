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

package vrouter_test

import (
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/hyperv_extension"
	"github.com/Juniper/contrail-windows-docker-driver/core/vrouter"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestVRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("vrouter_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "VRouter test suite",
		[]Reporter{junitReporter})
}

var _ = Describe("VRouter", func() {
	It("errors if extension is not Running", func() {
		ext := &hyperv_extension.HyperVExtensionSimulator{
			Enabled: false,
			Running: false,
		}
		vr := vrouter.NewHyperVvRouter(ext)
		err := vr.Initialize()
		Expect(err).To(HaveOccurred())
	})

	It("enables extension if it's not enabled", func() {
		ext := &hyperv_extension.HyperVExtensionSimulator{
			Enabled: false,
			Running: true,
		}
		vr := vrouter.NewHyperVvRouter(ext)
		err := vr.Initialize()
		Expect(err).ToNot(HaveOccurred())
		Expect(ext.IsEnabled()).To(BeTrue())
	})

	It("doesn't disable the extension if it's already enabled", func() {
		ext := &hyperv_extension.HyperVExtensionSimulator{
			Enabled: true,
			Running: true,
		}
		vr := vrouter.NewHyperVvRouter(ext)
		err := vr.Initialize()
		Expect(err).ToNot(HaveOccurred())
		Expect(ext.IsEnabled()).To(BeTrue())
	})

	Context("forced Forwarding Extension faults", func() {
		newGoodStartingExtension := func() hyperv_extension.HyperVExtensionSimulator {
			return hyperv_extension.HyperVExtensionSimulator{
				Enabled: false,
				Running: true,
			}
		}

		It("errors if after trying to enable the extension, it's not enabled", func() {
			ext := &hyperv_extension.HyperVExtensionFaultOnChangeSimulator{
				HyperVExtensionSimulator: newGoodStartingExtension(),
				IsEnabledAfter:           false,
				IsRunningAfter:           true,
			}
			vr := vrouter.NewHyperVvRouter(ext)
			err := vr.Initialize()
			Expect(err).To(HaveOccurred())
		})

		It("errors if after trying to enable the extension, it's not running", func() {
			ext := &hyperv_extension.HyperVExtensionFaultOnChangeSimulator{
				HyperVExtensionSimulator: newGoodStartingExtension(),
				IsEnabledAfter:           true,
				IsRunningAfter:           false,
			}
			vr := vrouter.NewHyperVvRouter(ext)
			err := vr.Initialize()
			Expect(err).To(HaveOccurred())
		})
	})

})
