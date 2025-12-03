// Copyright (c) 2025 The BFE Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openapi

import (
	"context"
	"fmt"

	"github.com/bfenetworks/service-controller/internal/alb/apis/product_pool"
	"github.com/bfenetworks/service-controller/internal/option/externalLB"
	util "github.com/bfenetworks/service-controller/internal/util"
	v1 "k8s.io/api/core/v1"
)

type ProductPoolname struct {
	Product  string
	Poolname string
}

type ProductPoolnameList []ProductPoolname

type AlbProvider struct {
	options *externalLB.Options
	client  *OpenApiClient
}

func NewAlbProvider(opts *externalLB.Options) *AlbProvider {
	return &AlbProvider{
		options: opts,
		client: NewOpenApiClient(
			opts.ApiServerAddr,
			opts.Token,
			opts.Timeout),
	}
}

func getInstances(ep *v1.Endpoints, portName string) []*product_pool.Instance {
	instances := make([]*product_pool.Instance, 0)

	for _, subset := range ep.Subsets {
		for _, p := range subset.Ports {
			if p.Name == portName {
				for _, addr := range subset.Addresses {
					instances = append(instances, &product_pool.Instance{
						Hostname: addr.IP, //addr.Hostname,
						IP:       addr.IP,
						Weight:   1,
						Ports:    map[string]int{"Default": int(p.Port)},
						Tags:     map[string]string{"key": "value"},
					})
				}
				break
			}
		}
	}

	return instances
}

func (p *AlbProvider) EnsureProductPool(ctx context.Context, product string, service *v1.Service,
	ep *v1.Endpoints, clusterName string) ([]string, error) {

	namespace := service.GetNamespace()
	name := service.GetName()
	poolNames := make([]string, 0, len(service.Spec.Ports))

	for _, port := range service.Spec.Ports {
		portName := port.Name
		if portName == "" {
			continue
		}

		pool := poolName(product, namespace, name, portName, clusterName)
		servers := getInstances(ep, portName)
		if len(servers) == 0 {
			util.HdlLogger.Info("product instance is empty, skip bfe api operation but record", "poolname", pool)
			poolNames = append(poolNames, pool)
			continue
		}

		param := &product_pool.UpsertParam{
			Name:      &pool,
			Instances: servers,
		}

		_, _, err := p.client.GetProductPool(product, pool)
		if err != nil {
			// pool doesn't exist yet. create it
			_, _, err := p.client.CreateProductPool(product, param)
			if err != nil {
				util.HdlLogger.Error(err, "failed to create product pool", "poolname", pool, "req", param)
				return poolNames, err
			} else {
				util.HdlLogger.Info("create product pool succ", "poolname", pool, "req", param)
				poolNames = append(poolNames, pool)
			}
		} else {
			// update it
			_, _, err := p.client.UpdateProductPool(product, param)
			if err != nil {
				util.HdlLogger.Error(err, "failed to update product pool", "poolname", pool, "req", param)
				return poolNames, err
			} else {
				util.HdlLogger.Info("update product pool succ", "poolname", pool, "req", param)
				poolNames = append(poolNames, pool)
			}
		}
	}
	return poolNames, nil
}

func (p *AlbProvider) DeleteProductPoolByList(ctx context.Context, poollist ProductPoolnameList) (ProductPoolnameList, error) {
	poolNames := make(ProductPoolnameList, 0, len(poollist))

	for _, pool := range poollist {
		e := p.client.DeleteProductPool(pool.Product, pool.Poolname)
		if e == nil {
			util.HdlLogger.Info("delete product pool succ", "poolname", pool.Poolname)
			poolNames = append(poolNames, pool)
		} else {
			util.HdlLogger.Error(e, "delete product pool", "poolname", pool.Poolname)
			return poolNames, e
		}
	}

	return poolNames, nil
}

func poolName(product string, namespace string, name string, portName string, clusterName string) string {
	if clusterName == "" {
		return fmt.Sprintf("%s.k8s_%s_%s_%s", product, namespace, name, portName)
	}
	return fmt.Sprintf("%s.k8s_%s_%s_%s_%s", product, namespace, name, portName, clusterName)
}
