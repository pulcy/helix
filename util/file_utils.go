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
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// EnsureDirectoryOf checks if the directory of the given file path exists and if not creates it.
// If such a path does exist, it checks if it is a directory, if not an error is returned.
func (s *sshClient) EnsureDirectoryOf(log zerolog.Logger, filePath string, perm os.FileMode) error {
	dirPath := filepath.Dir(filePath)
	if err := s.EnsureDirectory(log, dirPath, perm); err != nil {
		return maskAny(err)
	}
	return nil
}

// EnsureDirectory checks if a directory with given path exists and if not creates it.
// If such a path does exist, it checks if it is a directory, if not an error is returned.
func (s *sshClient) EnsureDirectory(log zerolog.Logger, dirPath string, perm os.FileMode) error {
	if _, err := s.Run(log, fmt.Sprintf("sh -c \"sudo mkdir -p %s && sudo chmod 0%o %s\"", dirPath, perm, dirPath), "", true); err != nil {
		return maskAny(err)
	}
	return nil
}

// UpdateFile compares the given content with the context of the file at the given filePath and
// if the content is different, the file is updated.
// If the file does not exist, it is created.
func (s *sshClient) UpdateFile(log zerolog.Logger, filePath string, content []byte, perm os.FileMode) error {
	if err := s.EnsureDirectoryOf(log, filePath, perm); err != nil {
		return maskAny(err)
	}
	if _, err := s.Run(log, fmt.Sprintf("sudo tee %s", filePath), string(content), true); err != nil {
		return maskAny(err)
	}
	if _, err := s.Run(log, fmt.Sprintf("sudo chmod 0%o %s", perm, filePath), "", true); err != nil {
		return maskAny(err)
	}
	return nil
}
