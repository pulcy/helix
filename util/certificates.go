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
	"crypto/x509"
	"crypto/x509/pkix"
	"io/ioutil"
	"os"
	"time"

	certificates "github.com/arangodb-helper/go-certificates"
)

const (
	caValidFor         = time.Hour * 24 * 365 * 10 // 10 years
	serverCertValidFor = time.Hour * 24 * 30       // 30 days
	adminCertValidFor  = time.Hour * 24 * 90       // 90 days

	CertFileMode = os.FileMode(0644)
	KeyFileMode  = os.FileMode(0600)
)

// NewServiceAccountCertificate tries to create a service account certificate pair.
// Returns cert, key, error
func NewServiceAccountCertificate(saPubPath, saKeyPath string) (string, string, error) {
	// Try to load from local file
	cert, key, err := loadCertificatePair(saPubPath, saKeyPath)
	if err == nil {
		// Got the files
	} else if os.IsNotExist(err) {
		opts := certificates.CreateCertificateOptions{
			Subject: &pkix.Name{
				CommonName:   "service-accounts",
				Organization: []string{"Helix"},
			},
			IsCA:        false,
			ValidFrom:   time.Now(),
			ValidFor:    caValidFor,
			ECDSACurve:  "P256",
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		}
		cert, key, err = certificates.CreateCertificate(opts, nil)
		if err != nil {
			return "", "", maskAny(err)
		}
		// Save files
		if err := saveCertificatePair(saPubPath, saKeyPath, cert, key); err != nil {
			return "", "", maskAny(err)
		}
	} else {
		// Some other error
		return "", "", maskAny(err)
	}
	return cert, key, nil
}

// CA is a Certificate Authority.
type CA struct {
	caCert string
	caKey  string
	ca     certificates.CA
}

// NewCA tries to load a CA from given path, if not found, creates a new one.
func NewCA(commonName string, caCertPath, caKeyPath string) (CA, error) {
	// Try to load from local file
	cert, key, err := loadCertificatePair(caCertPath, caKeyPath)
	if err == nil {
		// Got the files
	} else if os.IsNotExist(err) {
		// Files do not exist, create them
		opts := certificates.CreateCertificateOptions{
			Subject: &pkix.Name{
				CommonName:   commonName,
				Organization: []string{"Helix"},
			},
			IsCA:        true,
			ValidFrom:   time.Now(),
			ValidFor:    caValidFor,
			ECDSACurve:  "P256",
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		}
		cert, key, err = certificates.CreateCertificate(opts, nil)
		if err != nil {
			return CA{}, maskAny(err)
		}
		// Save files
		if err := saveCertificatePair(caCertPath, caKeyPath, cert, key); err != nil {
			return CA{}, maskAny(err)
		}
	} else {
		// Some other error
		return CA{}, maskAny(err)
	}
	result := CA{
		caCert: cert,
		caKey:  key,
	}
	result.ca, err = certificates.LoadCAFromPEM(cert, key)
	if err != nil {
		return CA{}, maskAny(err)
	}
	return result, nil
}

// Cert returns the CA certificate, as PEM encoded
func (ca *CA) Cert() string {
	return ca.caCert
}

// Key returns the CA private key, as PEM encoded
func (ca *CA) Key() string {
	return ca.caKey
}

// CreateServerCertificate creates a server certificates for the given client.
// Returns certificate, key, error.
func (ca *CA) CreateServerCertificate(commonName, orgName string, client SSHClient, additionalHosts ...string) (string, string, error) {
	opts := certificates.CreateCertificateOptions{
		Subject: &pkix.Name{
			CommonName:         commonName,
			Organization:       []string{orgName},
			OrganizationalUnit: []string{"Helix"},
		},
		Hosts: append([]string{client.GetAddress(), client.GetHostName()}, additionalHosts...),
		IsCA:  false,
		//IsClientAuth: false,
		ValidFrom:   time.Now(),
		ValidFor:    serverCertValidFor,
		ECDSACurve:  "P256",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	cert, key, err := certificates.CreateCertificate(opts, &ca.ca)
	if err != nil {
		return "", "", maskAny(err)
	}
	return cert, key, nil
}

// loadCertificatePair tries to load a cert+key from files with given paths.
func loadCertificatePair(certPath, keyPath string) (string, string, error) {
	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return "", "", err // do not mask
	}
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return "", "", err // do not mask
	}
	return string(cert), string(key), nil
}

// saveCertificatePair tries to save a cert+key tos files with given paths.
func saveCertificatePair(certPath, keyPath, cert, key string) error {
	if err := ioutil.WriteFile(certPath, []byte(cert), CertFileMode); err != nil {
		return maskAny(err)
	}
	if err := ioutil.WriteFile(keyPath, []byte(key), KeyFileMode); err != nil {
		return maskAny(err)
	}
	return nil
}
