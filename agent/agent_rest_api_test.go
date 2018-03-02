package agent_test

import (
	"github.com/Juniper/contrail-windows-docker-driver/agent"

	. "github.com/onsi/ginkgo"

	"fmt"
	"net/http"
	"net/http/httptest"
)

type httpHandler struct {
	statusToReturn int
	contentType string
	body string
}

func (h *httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Handler!")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("Handler!"))
}

var _ = Describe("AgentRestApi", func() {
	handler := &httpHandler{}
	testServer := httptest.NewServer(handler)
	defer testServer.Close()
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
				agentInstance.AddPort(
					vmUUID, vifUUID, ifName, mac,
					dockerID, ipAddress, vnUUID)
			})
		})
	})
})
