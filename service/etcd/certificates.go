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

package etcd

import (
	"time"

	certificates "github.com/arangodb-helper/go-certificates"
	"github.com/pulcy/helix/util"
)

const (
	caValidFor         = time.Hour * 24 * 365 * 10 // 10 years
	serverCertValidFor = time.Hour * 24 * 30       // 30 days
)

type ca struct {
	caCert string
	caKey  string
	ca     certificates.CA
}

// CreateCA initializes the CA structure.
func (ca *ca) CreateCA(commonName string, clientAuth bool) error {
	opts := certificates.CreateCertificateOptions{
		CommonName:   commonName,
		IsCA:         true,
		IsClientAuth: clientAuth,
		ValidFrom:    time.Now(),
		ValidFor:     caValidFor,
		ECDSACurve:   "P256",
	}
	cert, key, err := certificates.CreateCertificate(opts, nil)
	if err != nil {
		return maskAny(err)
	}
	ca.caCert = cert
	ca.caKey = key
	ca.ca, err = certificates.LoadCAFromPEM(cert, key)
	if err != nil {
		return maskAny(err)
	}
	return nil
}

// CreateServerCertificate creates a server certificates for the given client.
// Returns certificate, key, error.
func (ca *ca) CreateServerCertificate(client util.SSHClient) (string, string, error) {
	host := client.GetHost()
	opts := certificates.CreateCertificateOptions{
		CommonName:   host,
		Hosts:        []string{host},
		IsCA:         false,
		IsClientAuth: false,
		ValidFrom:    time.Now(),
		ValidFor:     serverCertValidFor,
		ECDSACurve:   "P256",
	}
	cert, key, err := certificates.CreateCertificate(opts, &ca.ca)
	if err != nil {
		return "", "", maskAny(err)
	}
	return cert, key, nil

}
