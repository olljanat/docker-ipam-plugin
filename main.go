package main

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	ipamApi "github.com/docker/go-plugins-helpers/ipam"
	"net"
)

// const socketAddress = "/run/docker/plugins/sdip.sock"
const pluginName = "sdip"
const localAddressSpace = "LOCAL"
const globalAddressSpace = "GLOBAL"

var scs = spew.ConfigState{Indent: "  "}

type pool struct {
	allocatedIPAddresses map[string]struct{}
}

type ipamDriver struct {
	pools map[string]*pool
}

func (i *ipamDriver) GetCapabilities() (*ipamApi.CapabilitiesResponse, error) {
	logrus.Infof("GetCapabilities called")
	return &ipamApi.CapabilitiesResponse{RequiresMACAddress: false}, nil
}

func (i *ipamDriver) GetDefaultAddressSpaces() (*ipamApi.AddressSpacesResponse, error) {
	logrus.Infof("GetDefaultAddressSpaces called")

	logrus.Infof("Returing response LocalDefaultAddressSpace: %s, GlobalDefaultAddressSpace: %s",
		localAddressSpace, globalAddressSpace)

	return &ipamApi.AddressSpacesResponse{LocalDefaultAddressSpace: localAddressSpace,
		GlobalDefaultAddressSpace: globalAddressSpace}, nil
}

func (i *ipamDriver) RequestPool(r *ipamApi.RequestPoolRequest) (*ipamApi.RequestPoolResponse, error) {
	// if !i.networkAllocated {
	// FixMe: Check if pool with same subnet already exists
	logrus.Infof("RequestPool called req:\n%+v\n", r)
	if r.Pool == "" {
		return &ipamApi.RequestPoolResponse{}, errors.New("Subnet is required")
	}

	rFormatted := scs.Sdump(r)
	logrus.Infof(rFormatted)

	// i.networkAllocated = true
	ipPool := &pool{
		allocatedIPAddresses: make(map[string]struct{}),
	}
	i.pools[r.Pool] = ipPool
	
	// Reserve subnet IP so gateway will get IP .1
	i.getNextIP(r.Pool)

	return &ipamApi.RequestPoolResponse{PoolID: r.Pool, Pool: r.Pool}, nil
	// }
	// return &ipamApi.RequestPoolResponse{}, errors.New("Pool Already Allocated")
}

func (i *ipamDriver) ReleasePool(r *ipamApi.ReleasePoolRequest) error {
	logrus.Infof("ReleasePool called req:\n%+v\n", r)

	rFormatted := scs.Sdump(r)
	logrus.Infof(rFormatted)

	// if r.PoolID == "1234" {
	logrus.Infof("Releasing Pool")
	// i.networkAllocated = false
	if i.pools[r.PoolID] != nil {
		//i.pools[r.PoolID].allocatedIPAddresses = make(map[string]struct{})
		delete(i.pools, r.PoolID)
	}
	//}
	return nil
}

func (i *ipamDriver) RequestAddress(r *ipamApi.RequestAddressRequest) (*ipamApi.RequestAddressResponse, error) {
	logrus.Infof("RequestAddress called req:\n%+v\n", r)

	rFormatted := scs.Sdump(r)
	logrus.Infof(rFormatted)

	addr := ""
	if r.Address != "" {
		addr = r.Address
	} else {
		if r.Options["RequestAddressType"] == "com.docker.network.gateway" || r.Options["com.docker.network.ipam.serial"] == "true" {
			addr = i.getNextIP(r.PoolID)
		} else {
			return &ipamApi.RequestAddressResponse{}, errors.New("IP is required")
		}
	}
	addr = fmt.Sprintf("%s/%s", addr, "24")
	logrus.Infof("Allocated IP %s", addr)
	return &ipamApi.RequestAddressResponse{Address: addr}, nil
}

func (i *ipamDriver) ReleaseAddress(r *ipamApi.ReleaseAddressRequest) error {
	logrus.Infof("ReleaseAddress called req:\n%+v\n", r)

	rFormatted := scs.Sdump(r)
	logrus.Infof(rFormatted)

	if i.pools[r.PoolID] != nil {
		delete(i.pools[r.PoolID].allocatedIPAddresses, r.Address)
		if _, ok := i.pools[r.PoolID].allocatedIPAddresses[r.Address]; !ok {
			logrus.Infof("IP %s Released from the Pool", r.Address)
		}
	}
	return nil
}

func (i *ipamDriver) getNextIP(pool string) string {
	ipAddr, ipNet, _ := net.ParseCIDR(pool)

	ret := ""
	for ip := ipAddr; ipNet.Contains(ip); incrementIP(ip) {
		if _, ok := i.pools[pool].allocatedIPAddresses[ip.String()]; !ok {
			ret = ip.String()
			i.pools[pool].allocatedIPAddresses[ret] = struct{}{}
			break
		}
	}
	return ret
}
func incrementIP(ip net.IP) {

	// length of IP is 16 bytes. This is because IPv6 address is 16 bytes.
	// For IPv4 , take the last octet and increment it by one.
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

func main() {
	logrus.Infof("Starting Docker IPAM Plugin")
	i := &ipamDriver{pools: make(map[string]*pool)}
	h := ipamApi.NewHandler(i)
	// logrus.Infof("Listening on socket %s", sdip)
	h.ServeUnix(pluginName, 0)
}
