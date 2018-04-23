package nal

import (
	"errors"
	"net"
	"time"

	"github.com/Juniper/contrail-windows-docker-driver/common"

	log "github.com/sirupsen/logrus"
)

type FlakyInterfaceGetter interface {
	InterfaceByName(ifname string) (*net.Interface, error)
}

type Nal struct {
	getter FlakyInterfaceGetter
}

type RealInterfaceGetter struct{}

func (*RealInterfaceGetter) InterfaceByName(ifname string) (*net.Interface, error) {
	return net.InterfaceByName(ifname)
}

func RealNal() Nal {
	return Nal{getter: &RealInterfaceGetter{}}
}

func isAutoconfigurationIP(ip net.IP) bool {
	return ip[0] == 169 && ip[1] == 254
}

func (nal Nal) WaitForInterface(ifname common.AdapterName) error {
	pollingStart := time.Now()
	for {
		queryStart := time.Now()
		iface, err := nal.getter.InterfaceByName(string(ifname))
		if err != nil {
			log.Warnf("Error when getting interface %s, but maybe it will appear soon: %s",
				ifname, err)
		} else {
			addrs, err := iface.Addrs()
			if err != nil {
				return err
			}

			// We print query time because it turns out that above operations actually take quite a
			// while (1-400ms), and the time depends (I think) on whether underlying interface
			// configs are being changed. For example, query usually takes ~10ms, but if it's about
			// to change, it can take up to 400ms. In other words, there must be some kind of mutex
			// there. This information could be useful for debugging.
			log.Debugf("Current %s addresses: %s. Query took %s", ifname,
				addrs, time.Since(queryStart))

			// We're essentialy waiting for adapter to reacquire IPv4 (that's how they do it
			// in Microsoft: https://github.com/Microsoft/hcsshim/issues/108)
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err == nil {
					ip = ip.To4()
					if ip != nil && !isAutoconfigurationIP(ip) {
						log.Debugf("Waited %s for IP reacquisition", time.Since(pollingStart))
						return nil
					}
				}
			}
		}

		if time.Since(pollingStart) > time.Millisecond*common.AdapterReconnectTimeout {
			return errors.New("Waited for net adapter to reconnect for too long.")
		}
		time.Sleep(time.Millisecond * common.AdapterPollingRate)
	}
}
