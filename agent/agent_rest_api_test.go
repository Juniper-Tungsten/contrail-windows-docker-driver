package agent_test

import (
	"github.com/Juniper/contrail-windows-docker-driver/agent"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
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
	testServer := httptest.NewServer(handler)
	var agentInstance agent.Agent

	vmUUID := "vmUUID"
	vifUUID := "vifUUID"
	ifName := "ifName"
	mac := "mac"
	dockerID := "dockerID"
	ipAddress := "ipAddress"
	vnUUID := "vnUUID"

	BeforeEach(func() {
		httpClient := &http.Client{Transport: &http.Transport{}}
		agentInstance = agent.NewAgentRestAPI(httpClient, &testServer.URL)
		fmt.Println(testServer.URL)
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
			})
		})
	})
})
