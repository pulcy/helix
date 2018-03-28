// Copyright (c) 2017 Pulcy.
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

package kubernetes

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pulcy/gluon/service"
	"github.com/pulcy/gluon/util"
)

const (
	cniPluginsSourcePath = "/home/core/bin/overlay/cni/bin"
	cniPluginsTargetPath = "/opt/cni/bin"
)

func linkCniBinaries(deps service.ServiceDependencies, flags *service.ServiceFlags) error {
	entries, err := ioutil.ReadDir(cniPluginsSourcePath)
	if err != nil {
		return maskAny(err)
	}
	if err := util.EnsureDirectory(cniPluginsTargetPath, 0755); err != nil {
		return maskAny(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		sourcePath := filepath.Join(cniPluginsSourcePath, e.Name())
		destPath := filepath.Join(cniPluginsTargetPath, e.Name())
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			deps.Logger.Debugf("Linking %s to %s", destPath, sourcePath)
			if err := os.Symlink(sourcePath, destPath); err != nil {
				return maskAny(err)
			}
		}
	}
	return nil
}
