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

package retry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/adapters/secondary/local_networking/hns/win_networking/retry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Retry")
}

var _ = Describe("Retry", func() {

	It("immediately returns if innerFunc returns no error", func() {
		counter := 0
		innerFunc := func() error {
			counter++
			return nil
		}
		err := retry.Retry(innerFunc, 10, time.Duration(0))
		Expect(err).ToNot(HaveOccurred())
		Expect(counter).To(Equal(1))
	})

	It("retries until innerFunc returns no error", func() {
		counter := 0
		innerFunc := func() error {
			counter++
			if counter == 5 {
				return nil
			} else {
				return errors.New("abcd")
			}
			return nil
		}
		err := retry.Retry(innerFunc, 10, time.Duration(0))
		Expect(err).ToNot(HaveOccurred())
		Expect(counter).To(Equal(5))
	})

	It("returns last inner error if innerFunc was never successful", func() {
		counter := 0
		innerFunc := func() error {
			counter++
			if counter < 10 {
				return errors.New("not last error")
			} else {
				return errors.New("last error")
			}
			return nil
		}
		err := retry.Retry(innerFunc, 10, time.Duration(0))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("last error"))
	})
})
