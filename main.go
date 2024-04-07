package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	ipamApi "github.com/docker/go-plugins-helpers/ipam"
)

const pluginName = "static"
const localAddressSpace = "LOCAL"
const globalAddressSpace = "GLOBAL"

var scs = spew.ConfigState{Indent: "  "}

type ipamDriver struct {
	v6gateway map[string]string
}

func (i *ipamDriver) GetCapabilities() (*ipamApi.CapabilitiesResponse, error) {
	return &ipamApi.CapabilitiesResponse{RequiresMACAddress: true}, nil
}

func (i *ipamDriver) GetDefaultAddressSpaces() (*ipamApi.AddressSpacesResponse, error) {
	return &ipamApi.AddressSpacesResponse{LocalDefaultAddressSpace: localAddressSpace,
		GlobalDefaultAddressSpace: globalAddressSpace}, nil
}

func (i *ipamDriver) RequestPool(r *ipamApi.RequestPoolRequest) (*ipamApi.RequestPoolResponse, error) {
	pool := ""
	v6gateway := ""
	if r.V6 {
		if r.Options["v6subnet"] == "" {
			return &ipamApi.RequestPoolResponse{}, errors.New("IPv6 subnet is required")
		}
		pool = r.Options["v6subnet"]

		if r.Options["v6gateway"] == "" {
			return &ipamApi.RequestPoolResponse{}, errors.New("IPv6 gateway is required")
		}
		v6gateway = r.Options["v6gateway"]
		i.v6gateway[pool] = v6gateway

	} else {
		if r.Pool == "" {
			return &ipamApi.RequestPoolResponse{}, errors.New("Subnet is required")
		}
		pool = r.Pool
	}

	return &ipamApi.RequestPoolResponse{PoolID: pool, Pool: pool}, nil
}

func (i *ipamDriver) ReleasePool(r *ipamApi.ReleasePoolRequest) error {
	return nil
}

func (i *ipamDriver) RequestAddress(r *ipamApi.RequestAddressRequest) (*ipamApi.RequestAddressResponse, error) {
	rFormatted := scs.Sdump(r)
	logrus.Infof(rFormatted)

	ip, ipnet, err := net.ParseCIDR(r.PoolID)
	if err != nil {
		return &ipamApi.RequestAddressResponse{}, err
	}

	address := r.Address
	if r.Address == "" && !strings.Contains(ip.String(), ":") {
		return &ipamApi.RequestAddressResponse{}, errors.New("IP is required")
	}

	if strings.Contains(ip.String(), ":") {
		if r.Options["RequestAddressType"] == "com.docker.network.gateway" {
			if i.v6gateway[r.PoolID] == "" {
				return &ipamApi.RequestAddressResponse{}, errors.New("Pool does not exist in driver database")
			}
			address = i.v6gateway[r.PoolID]
		} else {
			return &ipamApi.RequestAddressResponse{}, errors.New("IPv6 is required")
		}
	}

	mask, _ := ipnet.Mask.Size()
	addr := fmt.Sprintf("%s/%s", address, strconv.Itoa(mask))
	logrus.Infof("Parsed IP: %v", addr)
	return &ipamApi.RequestAddressResponse{Address: addr}, nil
}

func (i *ipamDriver) ReleaseAddress(r *ipamApi.ReleaseAddressRequest) error {
	return nil
}

func main() {
	logrus.Infof("Starting Docker IPAM Plugin")
	i := &ipamDriver{v6gateway: make(map[string]string)}
	h := ipamApi.NewHandler(i)
	h.ServeUnix(pluginName, 0)
}
