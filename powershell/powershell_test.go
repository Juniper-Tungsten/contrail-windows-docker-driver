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

// +build integration

package powershell_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/powershell"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPowershell(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Powershell")
}

var _ = Describe("Powershell wrapper", func() {

	var file *os.File
	BeforeEach(func() {
		var err error
		file, err = ioutil.TempFile("", "test-")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		file.Close()
	})

	It("returns only stdout if cmdlet didn't fail", func() {
		stdout, stderr, err := powershell.CallPowershell("Get-ChildItem", file.Name())
		Expect(err).ToNot(HaveOccurred())
		Expect(stdout).To(ContainSubstring(filepath.Base(file.Name())))
		Expect(stderr).To(Equal(""))
	})

	It("returns err and stderr if cmdlet failed", func() {
		stdout, stderr, err := powershell.CallPowershell("Get-ChildItem", file.Name()+"nonexisting")
		Expect(err).To(HaveOccurred())
		Expect(stdout).To(Equal(""))
		Expect(stderr).ToNot(Equal(""))
	})

	It("returns only stdout if wildcard matches", func() {
		wildcard := filepath.Join(filepath.Dir(file.Name()), "test-*")
		stdout, stderr, err := powershell.CallPowershell("Get-ChildItem", wildcard)
		Expect(err).ToNot(HaveOccurred())
		Expect(stdout).To(ContainSubstring(filepath.Base(file.Name())))
		Expect(stderr).To(Equal(""))
	})

	It("returns empty string if wildcard doesnt match, but no error or stderr", func() {
		wildcard := filepath.Join(filepath.Dir(file.Name()), "wontmatchanythinghopefully-*")
		stdout, stderr, err := powershell.CallPowershell("Get-ChildItem", wildcard)
		Expect(err).ToNot(HaveOccurred())
		Expect(stdout).To(Equal(""))
		Expect(stderr).To(Equal(""))
	})

})
