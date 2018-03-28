package main

import (
	"strings"

	"github.com/spf13/cobra"
)

const (
	hdr = ` _____       _              _    _      _ _      
|  __ \     | |            | |  | |    | (_)     
| |__) |   _| | ___ _   _  | |__| | ___| |___  __
|  ___/ | | | |/ __| | | | |  __  |/ _ \ | \ \/ /
| |   | |_| | | (__| |_| | | |  | |  __/ | |>  < 
|_|    \__,_|_|\___|\__, | |_|  |_|\___|_|_/_/\_\
                     __/ |                       
                    |___/                        
`
)

var (
	cmdVersion = &cobra.Command{
		Use: "version",
		Run: showVersion,
	}
)

func init() {
	cmdMain.AddCommand(cmdVersion)
}

func showVersion(cmd *cobra.Command, args []string) {
	for _, line := range strings.Split(hdr, "\n") {
		cliLog.Info().Msg(line)
	}
	cliLog.Info().Msgf("%s %s, build %s\n", cmdMain.Use, projectVersion, projectBuild)
}
