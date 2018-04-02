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
	"net/http"

	"github.com/ericchiang/k8s"
	"github.com/pkg/errors"
)

var (
	maskAny = errors.WithStack
)

// IsK8sConflict returns true if the given error is or is caused by a kubernetes conflict error.
func IsK8sConflict(err error) bool {
	if apiErr, ok := errors.Cause(err).(*k8s.APIError); ok {
		return apiErr.Code == http.StatusConflict && apiErr.Status != nil && apiErr.Status.GetReason() == "Conflict"
	}
	return false
}

// IsK8sAlreadyExists returns true if the given error is or is caused by a kubernetes not-found error.
func IsK8sAlreadyExists(err error) bool {
	if apiErr, ok := errors.Cause(err).(*k8s.APIError); ok {
		return apiErr.Code == http.StatusConflict && apiErr.Status != nil && apiErr.Status.GetReason() == "AlreadyExists"
	}
	return false
}

// IsK8sNotFound returns true if the given error is or is caused by a kubernetes not-found error.
func IsK8sNotFound(err error) bool {
	if apiErr, ok := errors.Cause(err).(*k8s.APIError); ok {
		return apiErr.Code == http.StatusNotFound
	}
	return false
}
