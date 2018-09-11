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

// +build unit

package controller_rest_test

import (
	. "github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest"
	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/controller_rest/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller Adapter", func() {
	var fakeAPIClient = api.NewFakeApiClient()

	Describe("initializing controller adapter", func() {
		Context("with noauth constructor", func() {
			It("creates admin project if it doesn't exist", func() {
				controller, err := NewControllerInsecureAdapter(fakeAPIClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(controller).ToNot(BeNil())
				adminProject, err := controller.GetProject(DomainName, AdminProject)
				Expect(err).ToNot(HaveOccurred())
				Expect(adminProject).ToNot(BeNil())
			})
			It("doesn't fail if admin project exists", func() {
				CreateTestProject(fakeAPIClient, DomainName, AdminProject)
				controller, err := NewControllerInsecureAdapter(fakeAPIClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(controller).ToNot(BeNil())
			})
		})
	})
})
