package client

import (
	"github.com/bakito/kubexporter/pkg/types"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type ApiClient struct {
	RestConfig      *rest.Config
	Client          dynamic.Interface
	Mapper          *restmapper.DeferredDiscoveryRESTMapper
	DiscoveryClient *discovery.DiscoveryClient
}

func NewApiClient(config *types.Config) (*ApiClient, error) {
	rc, err := config.RestConfig()
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(rc)
	if err != nil {
		return nil, err
	}

	dcl, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dcl))
	return &ApiClient{RestConfig: rc, Client: client, Mapper: mapper, DiscoveryClient: dcl}, nil
}
