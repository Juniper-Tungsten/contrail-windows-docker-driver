package agent_test

import (
	"github.com/Juniper/contrail-windows-docker-driver/agent"

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
	handler := &httpHandler{}
	var testServer *httptest.Server
	var agentInstance agent.Agent

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
		testServer = httptest.NewServer(handler)
		httpClient := &http.Client{Transport: &http.Transport{}}
		agentInstance = agent.NewAgentRestAPI(httpClient, &testServer.URL)
	})

	AfterEach(func() {
		testServer.Close()
	})

	Describe("AddPort method", func() {
		Context("When AddPort is correctly invoked", func() {
			It("should send correct JSON request", func() {
				handler.response.status = http.StatusOK
				handler.response.body = "{}"

				err := agentInstance.AddPort(
					vmUUID, vifUUID, ifName, mac,
					dockerID, ipAddress, vnUUID)

				Expect(handler.request.method).To(Equal("POST"))
				Expect(handler.request.path).To(Equal("/port"))
				Expect(err).To(BeNil())

				var requestMessage agent.PortRequestMsg
				err = json.Unmarshal(handler.request.body, &requestMessage)
				Expect(err).To(BeNil())

				Expect(requestMessage.VmUUID).To(Equal(vmUUID))
				Expect(requestMessage.VifUUID).To(Equal(vifUUID))
				Expect(requestMessage.IfName).To(Equal(ifName))
				Expect(requestMessage.Mac).To(Equal(mac))
				Expect(requestMessage.DockerID).To(Equal(dockerID))
				Expect(requestMessage.IpAddress).To(Equal(ipAddress))
				Expect(requestMessage.VnUUID).To(Equal(vnUUID))
				Expect(requestMessage.Ipv6).To(Equal(""))
				Expect(requestMessage.Type).To(Equal(1))
				Expect(requestMessage.RxVlanId).To(Equal(-1))
				Expect(requestMessage.TxVlanId).To(Equal(-1))
				Expect(requestMessage.VmProjectId).To(Equal(""))
			})
		})

		Context("When AddPort is correctly invoked and server returns error", func() {
			It("should return error", func() {
				handler.response.status = http.StatusTeapot
				handler.response.body = "{}"

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
				handler.response.status = http.StatusOK
				handler.response.body = "{}"

				err := agentInstance.DeletePort(vifUUID)

				Expect(handler.request.method).To(Equal("DELETE"))
				Expect(handler.request.path).To(Equal("/port/" + vifUUID))
				Expect(err).To(BeNil())
			})
		})

		Context("When DeletePort is correctly invoked and server returns error", func() {
			It("should return error", func() {
				handler.response.status = http.StatusTeapot
				handler.response.body = "{}"

				err := agentInstance.DeletePort(vifUUID)
				Expect(err).NotTo(BeNil())
			})
		})
	})
})
