// Copyright (c) 2016 Pulcy.
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

package main

import (
	"os"
	"strconv"
)

func defaultEtcdSecureClients() bool {
	return boolFromEnv("GLUON_ETCD_SECURE_CLIENTS", false)
}

func defaultKubernetesAPIDNSName() string {
	return os.Getenv("GLUON_K8S_API_DNS_NAME")
}

func boolFromEnv(key string, defaultValue bool) bool {
	x := os.Getenv(key)
	if x == "" {
		return defaultValue
	}
	result, _ := strconv.ParseBool(x)
	return result
}
