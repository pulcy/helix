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
	"bytes"
	"html/template"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type TemplateConfigurator func(*template.Template)

// Render updates the given destinationPath according to the given template and options.
func (s *sshClient) Render(log zerolog.Logger, templateData, destinationPath string, options interface{}, destinationFileMode os.FileMode, config ...TemplateConfigurator) error {
	// parse template
	var tmpl *template.Template
	tmpl = template.New("name")
	funcMap := template.FuncMap{
		"escape": escape,
		"quote":  strconv.Quote,
	}
	tmpl.Funcs(funcMap)
	for _, c := range config {
		c(tmpl)
	}
	if _, err := tmpl.Parse(templateData); err != nil {
		return maskAny(err)
	}
	// execute template to buffer
	buf := &bytes.Buffer{}
	tmpl.Funcs(funcMap)
	if err := tmpl.Execute(buf, options); err != nil {
		return maskAny(err)
	}

	// Update file
	if err := s.UpdateFile(log, destinationPath, buf.Bytes(), destinationFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}

func escape(s string) string {
	s = strconv.Quote(s)
	return s[1 : len(s)-1]
}
