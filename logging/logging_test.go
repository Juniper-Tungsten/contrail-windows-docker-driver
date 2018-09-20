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

// +build unit

package logging_test

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/Juniper/contrail-windows-docker-driver/logging"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("logging_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Logging test suite",
		[]Reporter{junitReporter})
}

var _ = Describe("Logging tests", func() {
	packageTag := "LoggingTest"
	Context("HTTP request log text test", func() {
		It("returns error message when given not response nor request", func() {
			ret := logging.HTTPMessage(packageTag, "abcdefgh")
			Expect(ret).To(Equal(logging.WrongHTTPMessageParameter))
		})

		It("returns error message when request is nil", func() {
			var request *http.Request
			request = nil
			ret := logging.HTTPMessage(packageTag, request)
			Expect(ret).To(Equal(logging.EmptyRequestResponseMessage))
		})

		It("returns information when request body is nil", func() {
			request, _ := http.NewRequest("GET", "", nil)

			ret := logging.HTTPMessage(packageTag, request)

			Expect(ret).To(ContainSubstring(logging.EmptyHTTPBody))
		})

		It("shows correct body when it exists", func() {
			request, _ := http.NewRequest("GET", "", ioutil.NopCloser(bytes.NewBufferString("Ala ma kota.")))

			ret := logging.HTTPMessage(packageTag, request)

			Expect(ret).To(ContainSubstring("Ala ma kota."))
		})

		It("preserves body in request", func() {
			request, _ := http.NewRequest("GET", "", ioutil.NopCloser(bytes.NewBufferString("Ala ma kota.")))

			ret := logging.HTTPMessage(packageTag, request)
			ret2 := logging.HTTPMessage(packageTag, request)

			Expect(ret).To(ContainSubstring("Ala ma kota."))
			Expect(ret).To(Equal(ret2))
		})

		It("shows correct method", func() {
			requestGet, _ := http.NewRequest("GET", "", nil)
			requestPost, _ := http.NewRequest("POST", "", nil)
			requestDelete, _ := http.NewRequest("DELETE", "", nil)

			retGet := logging.HTTPMessage(packageTag, requestGet)
			retPost := logging.HTTPMessage(packageTag, requestPost)
			retDelete := logging.HTTPMessage(packageTag, requestDelete)

			Expect(retGet).To(ContainSubstring("GET"))
			Expect(retPost).To(ContainSubstring("POST"))
			Expect(retDelete).To(ContainSubstring("DELETE"))
		})

		It("shows correct header", func() {
			request, _ := http.NewRequest("GET", "", nil)
			request.Header.Set("Content-Type", "application/json")

			ret := logging.HTTPMessage(packageTag, request)

			Expect(ret).To(ContainSubstring("application/json"))
		})

		It("shows correct url", func() {
			request1, _ := http.NewRequest("GET", "http://abc.def/path1", nil)
			request2, _ := http.NewRequest("GET", "http://abc.def/path2", nil)
			request3, _ := http.NewRequest("GET", "http://abc.def/path3", nil)

			ret1 := logging.HTTPMessage(packageTag, request1)
			ret2 := logging.HTTPMessage(packageTag, request2)
			ret3 := logging.HTTPMessage(packageTag, request3)

			Expect(ret1).To(ContainSubstring("[http://abc.def/path1]"))
			Expect(ret2).To(ContainSubstring("[http://abc.def/path2]"))
			Expect(ret3).To(ContainSubstring("[http://abc.def/path3]"))
		})

		It("shows correct tag", func() {
			request, _ := http.NewRequest("GET", "", nil)
			request.Header.Set("Content-Type", "application/json")

			ret := logging.HTTPMessage(packageTag, request)

			Expect(ret).To(ContainSubstring("[%s]", packageTag))
		})

	})

	Context("HTTP response log text test", func() {
		client := &http.Client{}
		server := testServer(make(chan interface{}))
		testHeader := http.Header{}
		testHeader.Add("headerTxt", "Ala Ma Kota W Glowie")

		It("returns error message when response is nil", func() {
			var response *http.Response
			response = nil
			ret := logging.HTTPMessage(packageTag, response)

			Expect(ret).To(Equal(logging.EmptyRequestResponseMessage))
		})

		It("returns information when response body is nil", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)
			request, _ := http.NewRequest("GET", "http://127.0.0.1:9091/", nil)

			response, _ := client.Do(request)
			response.Body = nil

			ret := logging.HTTPMessage(packageTag, response)

			Expect(ret).To(ContainSubstring("Body is empty."))
		})

		It("shows correct body when it exists", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bodyTest"),
					ghttp.RespondWith(http.StatusOK, "Ala Ma Kota"),
				),
			)
			request, _ := http.NewRequest("GET", "http://127.0.0.1:9091/bodyTest", nil)

			response, _ := client.Do(request)

			ret := logging.HTTPMessage(packageTag, response)

			Expect(ret).To(ContainSubstring("Body: Ala Ma Kota"))
		})

		It("preserves body in response", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bodyTest"),
					ghttp.RespondWith(http.StatusOK, "Ala Ma Kota"),
				),
			)
			request, _ := http.NewRequest("GET", "http://127.0.0.1:9091/bodyTest", nil)

			response, _ := client.Do(request)

			ret := logging.HTTPMessage(packageTag, response)
			ret2 := logging.HTTPMessage(packageTag, response)

			Expect(ret).To(ContainSubstring("Body: Ala Ma Kota"))
			Expect(ret).To(Equal(ret2))
		})

		It("shows correct header", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/headerTest"),
					ghttp.RespondWith(http.StatusOK, "", testHeader),
				),
			)
			request, _ := http.NewRequest("GET", "http://127.0.0.1:9091/headerTest", nil)

			response, _ := client.Do(request)
			ret := logging.HTTPMessage(packageTag, response)

			Expect(ret).To(ContainSubstring("\"Headertxt\":[\"Ala Ma Kota W Glowie\"]"))
		})

		It("shows correct status", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/status/200"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/status/404"),
					ghttp.RespondWith(http.StatusNotFound, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/status/500"),
					ghttp.RespondWith(http.StatusInternalServerError, ""),
				),
			)

			request200, _ := http.NewRequest("GET", "http://127.0.0.1:9091/status/200", nil)
			request404, _ := http.NewRequest("GET", "http://127.0.0.1:9091/status/404", nil)
			request500, _ := http.NewRequest("GET", "http://127.0.0.1:9091/status/500", nil)

			response200, _ := client.Do(request200)
			response404, _ := client.Do(request404)
			response500, _ := client.Do(request500)

			ret200 := logging.HTTPMessage(packageTag, response200)
			ret404 := logging.HTTPMessage(packageTag, response404)
			ret500 := logging.HTTPMessage(packageTag, response500)

			Expect(ret200).To(ContainSubstring("Status: %d", http.StatusOK))
			Expect(ret404).To(ContainSubstring("Status: %d", http.StatusNotFound))
			Expect(ret500).To(ContainSubstring("Status: %d", http.StatusInternalServerError))
		})

		It("shows correct tag", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			request, _ := http.NewRequest("GET", "http://127.0.0.1:9091/", nil)

			response, _ := client.Do(request)

			ret := logging.HTTPMessage(packageTag, response)

			Expect(ret).To(ContainSubstring("[%s]", packageTag))
		})

	})
})

func testServer(recv chan interface{}) *ghttp.Server {
	server := ghttp.NewUnstartedServer()
	listener, _ := net.Listen("tcp", "127.0.0.1:9091")
	server.HTTPTestServer.Listener = listener

	server.Start()
	return server
}
