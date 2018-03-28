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

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
)

var (
	cmdMain = &cobra.Command{
		Use:   "helix",
		Short: "Helix bootstraps Kubernetes cluster",
		Long:  "Helix creates a highly available ETCD cluster and bootstraps Kubernetes",
		Run:   showUsage,
	}
	cliLog zerolog.Logger
)

func init() {
	cliLog = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
}

func main() {
	cmdMain.Execute()
}

func showUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func Exitf(format string, args ...interface{}) {
	cliLog.Error().Msgf(format, args...)
	os.Exit(1)
}

func assertArgIsSet(arg, argKey string) {
	if arg == "" {
		Exitf("%s must be set\n", argKey)
	}
}
