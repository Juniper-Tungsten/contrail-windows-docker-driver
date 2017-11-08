package hns

import (
	"encoding/json"

	"github.com/Microsoft/hcsshim"
	log "github.com/sirupsen/logrus"
	"github.com/Juniper/contrail-windows-docker-driver/common"
)

func CreateHNSNetwork(configuration *hcsshim.HNSNetwork) (string, error) {
	log.Infoln("Creating HNS network")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config:", string(configBytes))

	response, err := hcsshim.HNSNetworkRequest("POST", "", string(configBytes))
	if err != nil {
		log.Errorln(err)
		return "", err
	}

	// When the first HNS network is created, a vswitch is also created and attached to
	// specified network adapter. This adapter will temporarily lose network connectivity
	// while it reacquires IPv4. We need to wait for it.
	// https://github.com/Microsoft/hcsshim/issues/108
	if err := common.WaitForInterface(common.HNSTransparentInterfaceName); err != nil {
		log.Errorln(err)
		return "", err
	}

	log.Infoln("Created HNS network with ID:", response.Id)

	return response.Id, nil
}

func DeleteHNSNetwork(hnsID string) error {
	log.Infoln("Deleting HNS network", hnsID)

	toDelete, err := GetHNSNetwork(hnsID)
	if err != nil {
		log.Errorln(err)
		return err
	}

	networks, err := ListHNSNetworks()
	if err != nil {
		log.Errorln(err)
		return err
	}

	adapterStillInUse := false
	for _, network := range networks {
		if network.Id != toDelete.Id &&
			network.NetworkAdapterName == toDelete.NetworkAdapterName {
			adapterStillInUse = true
			break
		}
	}

	_, err = hcsshim.HNSNetworkRequest("DELETE", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}

	if !adapterStillInUse {
		// If the last network that uses an adapter is deleted, then the underlying vswitch is
		// also deleted. During this period, the adapter will temporarily lose network
		// connectivity while it reacquires IPv4. We need to wait for it.
		// https://github.com/Microsoft/hcsshim/issues/95
		if err := common.WaitForInterface(
			common.AdapterName(toDelete.NetworkAdapterName)); err != nil {
			log.Errorln(err)
			return err
		}
	}

	return nil
}

func ListHNSNetworks() ([]hcsshim.HNSNetwork, error) {
	log.Infoln("Listing HNS networks")
	nets, err := hcsshim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return nets, nil
}

func GetHNSNetwork(hnsID string) (*hcsshim.HNSNetwork, error) {
	log.Infoln("Getting HNS network", hnsID)
	net, err := hcsshim.HNSNetworkRequest("GET", hnsID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return net, nil
}

func GetHNSNetworkByName(name string) (*hcsshim.HNSNetwork, error) {
	log.Infoln("Getting HNS network by name:", name)
	nets, err := hcsshim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	for _, n := range nets {
		if n.Name == name {
			return &n, nil
		}
	}
	return nil, nil
}

func CreateHNSEndpoint(configuration *hcsshim.HNSEndpoint) (string, error) {
	log.Infoln("Creating HNS endpoint")
	configBytes, err := json.Marshal(configuration)
	if err != nil {
		log.Errorln(err)
		return "", err
	}
	log.Debugln("Config: ", string(configBytes))
	response, err := hcsshim.HNSEndpointRequest("POST", "", string(configBytes))
	if err != nil {
		return "", err
	}
	log.Infoln("Created HNS endpoint with ID:", response.Id)
	return response.Id, nil
}

func DeleteHNSEndpoint(endpointID string) error {
	log.Infoln("Deleting HNS endpoint", endpointID)
	_, err := hcsshim.HNSEndpointRequest("DELETE", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return err
	}
	return nil
}

func GetHNSEndpoint(endpointID string) (*hcsshim.HNSEndpoint, error) {
	log.Infoln("Getting HNS endpoint", endpointID)
	endpoint, err := hcsshim.HNSEndpointRequest("GET", endpointID, "")
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	return endpoint, nil
}

func GetHNSEndpointByName(name string) (*hcsshim.HNSEndpoint, error) {
	log.Infoln("Getting HNS endpoint by name:", name)
	eps, err := hcsshim.HNSListEndpointRequest()
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	for _, ep := range eps {
		if ep.Name == name {
			return &ep, nil
		}
	}
	return nil, nil
}

func ListHNSEndpoints() ([]hcsshim.HNSEndpoint, error) {
	endpoints, err := hcsshim.HNSListEndpointRequest()
	if err != nil {
		return nil, err
	}
	return endpoints, nil
}

func ListHNSEndpointsOfNetwork(netID string) ([]hcsshim.HNSEndpoint, error) {
	eps, err := ListHNSEndpoints()
	if err != nil {
		return nil, err
	}
	var epsInNetwork []hcsshim.HNSEndpoint
	for _, ep := range eps {
		if ep.VirtualNetwork == netID {
			epsInNetwork = append(epsInNetwork, ep)
		}
	}
	return epsInNetwork, nil
}
