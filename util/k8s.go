// Copyright (c) 2018 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"

	"github.com/ericchiang/k8s/util/intstr"

	"github.com/ericchiang/k8s"
)

// CreateOrUpdate creates or updates a given resource.
func CreateOrUpdate(ctx context.Context, client *k8s.Client, req k8s.Resource, options ...k8s.Option) error {
	if err := client.Create(ctx, req, options...); err == nil {
		return nil
	} else if !IsK8sAlreadyExists(err) {
		return maskAny(err)
	}
	// Exists, update it
	if err := client.Update(ctx, req, options...); err != nil {
		return maskAny(err)
	}
	return nil
}

// IntOrStringI returns an IntOrString filled with an int.
func IntOrStringI(i int32) *intstr.IntOrString {
	return &intstr.IntOrString{
		IntVal: k8s.Int32(i),
	}
}

// IntOrStringS returns an IntOrString filled with a string.
func IntOrStringS(s string) *intstr.IntOrString {
	return &intstr.IntOrString{
		StrVal: k8s.String(s),
	}
}
