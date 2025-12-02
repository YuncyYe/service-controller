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

package filter

import (
	"github.com/bfenetworks/service-controller/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	BfenetworksAnnotationPrefix = "k8s.bfenetworks.com/"
)

func isBfenetworksTargetService(service client.Object) bool {
	sname := service.GetName()
	labels := service.GetLabels()
	if labels != nil {
		_, ipok := labels["bfe-product"]
		if ipok {
			return true
		}
	}
	util.HdlLogger.Info("bfe-product label does not present for k8s service", "sname", sname)

	return false
}

func LabelFilter() predicate.Funcs {
	funcs := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		labels := obj.GetLabels()
		if labels == nil {
			return false
		}

		return isBfenetworksTargetService(obj)
	})

	return funcs
}
