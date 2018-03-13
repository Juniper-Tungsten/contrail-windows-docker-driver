package agent_test

import (
	"net/url"

	"github.com/Juniper/contrail-windows-docker-driver/agent"
	"github.com/Juniper/contrail-windows-docker-driver/driver"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

type httpHandler struct {
	response struct {
		status int
		body   string
	}

	request struct {
		method string
		path   string
		body   []byte
	}
}

func (h *httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(h.response.status)
	res.Header().Set("Content-Type", "application/json")
	res.Write([]byte(h.response.body))

	h.request.method = req.Method
	h.request.path = req.URL.Path
	h.request.body, _ = ioutil.ReadAll(req.Body)
}

var _ = Describe("AgentRestApi", func() {
	mockHandler := &httpHandler{}
	var testServer *httptest.Server
	var agentInstance driver.Agent

	const (
		vmUUID    = "vmUUID"
		vifUUID   = "vifUUID"
		ifName    = "ifName"
		mac       = "mac"
		dockerID  = "dockerID"
		ipAddress = "ipAddress"
		vnUUID    = "vnUUID"
	)

	BeforeEach(func() {
		testServer = httptest.NewServer(mockHandler)
		httpClient := &http.Client{Transport: &http.Transport{}}
		serverUrl, _ := url.Parse(testServer.URL)
		agentInstance = agent.NewAgentRestAPI(httpClient, serverUrl)
	})

	AfterEach(func() {
		testServer.Close()
	})

	Describe("AddPort method", func() {
		Context("When AddPort is correctly invoked", func() {
			It("should send correct http request", func() {
				mockHandler.response.status = http.StatusOK
				mockHandler.response.body = "{}"

				err := agentInstance.AddPort(
					vmUUID, vifUUID, ifName, mac,
					dockerID, ipAddress, vnUUID)

				Expect(mockHandler.request.method).To(Equal("POST"))
				Expect(mockHandler.request.path).To(Equal("/port"))
				Expect(err).To(BeNil())
			})

			It("should send correct JSON request body", func() {
				mockHandler.response.status = http.StatusOK
				mockHandler.response.body = "{}"

				agentInstance.AddPort(
					vmUUID, vifUUID, ifName, mac,
					dockerID, ipAddress, vnUUID)

				var requestMessage agent.PortRequestMsg
				err := json.Unmarshal(mockHandler.request.body, &requestMessage)
				Expect(err).To(BeNil())

				Expect(requestMessage.VmUUID).To(Equal(vmUUID))
				Expect(requestMessage.VifUUID).To(Equal(vifUUID))
				Expect(requestMessage.IfName).To(Equal(ifName))
				Expect(requestMessage.Mac).To(Equal(mac))
				Expect(requestMessage.DockerID).To(Equal(dockerID))
				Expect(requestMessage.IpAddress).To(Equal(ipAddress))
				Expect(requestMessage.VnUUID).To(Equal(vnUUID))
				Expect(requestMessage.Ipv6).To(Equal(""))
				Expect(requestMessage.Type).To(Equal(0)) // 0 - vm port type
				Expect(requestMessage.RxVlanId).To(Equal(-1))
				Expect(requestMessage.TxVlanId).To(Equal(-1))
				Expect(requestMessage.VmProjectId).To(Equal(""))
			})
		})

		Context("When AddPort is correctly invoked and server returns error", func() {
			It("should return error", func() {
				mockHandler.response.status = http.StatusTeapot
				mockHandler.response.body = "{}"

				err := agentInstance.AddPort(
					vmUUID, vifUUID, ifName, mac,
					dockerID, ipAddress, vnUUID)

				Expect(err).NotTo(BeNil())
			})
		})
	})

	Describe("DeletePort method", func() {
		Context("When DeletePort is correctly invoked", func() {
			It("should send correct JSON request", func() {
				mockHandler.response.status = http.StatusOK
				mockHandler.response.body = "{}"

				err := agentInstance.DeletePort(vifUUID)

				Expect(mockHandler.request.method).To(Equal("DELETE"))
				Expect(mockHandler.request.path).To(Equal("/port/" + vifUUID))
				Expect(err).To(BeNil())
			})
		})

		Context("When DeletePort is correctly invoked and server returns error", func() {
			It("should return error", func() {
				mockHandler.response.status = http.StatusTeapot
				mockHandler.response.body = "{}"

				err := agentInstance.DeletePort(vifUUID)
				Expect(err).NotTo(BeNil())
			})
		})
	})
})
