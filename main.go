package main

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	ipamApi "github.com/docker/go-plugins-helpers/ipam"
)

const pluginName = "static"
const localAddressSpace = "LOCAL"
const globalAddressSpace = "GLOBAL"

var scs = spew.ConfigState{Indent: "  "}

type ipamDriver struct {
}

func (i *ipamDriver) GetCapabilities() (*ipamApi.CapabilitiesResponse, error) {
	return &ipamApi.CapabilitiesResponse{RequiresMACAddress: true}, nil
}

func (i *ipamDriver) GetDefaultAddressSpaces() (*ipamApi.AddressSpacesResponse, error) {
	return &ipamApi.AddressSpacesResponse{LocalDefaultAddressSpace: localAddressSpace,
		GlobalDefaultAddressSpace: globalAddressSpace}, nil
}

func (i *ipamDriver) RequestPool(r *ipamApi.RequestPoolRequest) (*ipamApi.RequestPoolResponse, error) {
	if r.Pool == "" {
		return &ipamApi.RequestPoolResponse{}, errors.New("Subnet is required")
	}

	return &ipamApi.RequestPoolResponse{PoolID: r.Pool, Pool: r.Pool}, nil
}

func (i *ipamDriver) ReleasePool(r *ipamApi.ReleasePoolRequest) error {
	return nil
}

func (i *ipamDriver) RequestAddress(r *ipamApi.RequestAddressRequest) (*ipamApi.RequestAddressResponse, error) {

	if r.Address == "" {
		return &ipamApi.RequestAddressResponse{}, errors.New("IP is required")
	}

	// FixMe: Do not hardcode subnet mask
	addr := fmt.Sprintf("%s/%s", r.Address, "24")
	return &ipamApi.RequestAddressResponse{Address: addr}, nil
}

func (i *ipamDriver) ReleaseAddress(r *ipamApi.ReleaseAddressRequest) error {
	return nil
}

func main() {
	logrus.Infof("Starting Docker IPAM Plugin")
	i := &ipamDriver{}
	h := ipamApi.NewHandler(i)
	h.ServeUnix(pluginName, 0)
}
