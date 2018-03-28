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

package util

import (
	"bytes"
	"io"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHClient interface {
	io.Closer
	GetHost() string
	Run(log zerolog.Logger, command, stdin string, quiet bool) (string, error)

	// EnsureDirectoryOf checks if the directory of the given file path exists and if not creates it.
	// If such a path does exist, it checks if it is a directory, if not an error is returned.
	EnsureDirectoryOf(log zerolog.Logger, filePath string, perm os.FileMode) error
	// EnsureDirectory checks if a directory with given path exists and if not creates it.
	// If such a path does exist, it checks if it is a directory, if not an error is returned.
	EnsureDirectory(log zerolog.Logger, dirPath string, perm os.FileMode) error
	// UpdateFile compares the given content with the context of the file at the given filePath and
	// if the content is different, the file is updated.
	// If the file does not exist, it is created.
	UpdateFile(log zerolog.Logger, filePath string, content []byte, perm os.FileMode) error
	// Render updates the given destinationPath according to the given template and options.
	Render(log zerolog.Logger, templateData, destinationPath string, options interface{}, destinationFileMode os.FileMode, config ...TemplateConfigurator) error
}

type sshClient struct {
	client *ssh.Client
	host   string
	dryRun bool
}

// DialSSH creates a new SSH connection to the given user on the given host.
func DialSSH(userName, host string, dryRun bool) (SSHClient, error) {
	// To authenticate with the remote server you must pass at least one
	// implementation of AuthMethod via the Auth field in ClientConfig.
	config := &ssh.ClientConfig{
		User:            userName,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	var sshAgent agent.Agent
	if agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		sshAgent = agent.NewClient(agentConn)
		config.Auth = append(config.Auth, ssh.PublicKeysCallback(sshAgent.Signers))
	} else {
		return nil, maskAny(err)
	}

	addr := net.JoinHostPort(host, "22")
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, maskAny(err)
	}

	return &sshClient{
		client: client,
		host:   host,
		dryRun: dryRun,
	}, nil
}

func (s *sshClient) GetHost() string {
	return s.host
}

func (s *sshClient) Close() error {
	return maskAny(s.client.Close())
}

func (s *sshClient) Run(log zerolog.Logger, command, stdin string, quiet bool) (string, error) {
	if s.dryRun {
		log.Info().Msgf("Will run: %s", command)
		return "", nil
	}

	var stdOut, stdErr bytes.Buffer

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := s.client.NewSession()
	if err != nil {
		return "", maskAny(err)
	}
	defer session.Close()

	session.Stdout = &stdOut
	session.Stderr = &stdErr

	if stdin != "" {
		session.Stdin = strings.NewReader(stdin)
	}

	if err := session.Run(command); err != nil {
		if !quiet {
			log.Error().Msgf("SSH failed: %s", command)
		}
		return "", errors.Wrapf(err, stdErr.String())
	}

	out := stdOut.String()
	out = strings.TrimSuffix(out, "\n")
	return out, nil
}
