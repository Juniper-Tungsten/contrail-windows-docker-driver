package hns

import "github.com/Microsoft/hcsshim"

type Network = hcsshim.HNSNetwork
type Endpoint = hcsshim.HNSEndpoint

type HcsShim interface {
	HNSNetworkRequest(method, path, request string) (*Network, error)
	HNSEndpointRequest(method, endpointID, request string) (*Endpoint, error)
	HNSListEndpointRequest() ([]hcsshim.HNSEndpoint, error)
	HNSListNetworkRequest(method, path, request string) ([]Network, error)
}

type RealHcsShim struct{}

func (*RealHcsShim) HNSNetworkRequest(method, path, request string) (*Network, error) {
	return hcsshim.HNSNetworkRequest(method, path, request)
}

func (*RealHcsShim) HNSEndpointRequest(method, path, request string) (*Endpoint, error) {
	return hcsshim.HNSEndpointRequest(method, path, request)
}

func (*RealHcsShim) HNSListEndpointRequest() ([]Endpoint, error) {
	return hcsshim.HNSListEndpointRequest()
}

func (*RealHcsShim) HNSListNetworkRequest(method, path, request string) ([]Network, error) {
	return hcsshim.HNSListNetworkRequest(method, path, request)
}
