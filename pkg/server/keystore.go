/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"io/ioutil"
	"path"

	"k8s.io/kops/pkg/pki"
)

type keystore struct {
	keys map[string]keystoreEntry
}

type keystoreEntry struct {
	certificate *pki.Certificate
	key         *pki.PrivateKey
}

var _ pki.Keystore = keystore{}

func (k keystore) FindKeypair(name string) (*pki.Certificate, *pki.PrivateKey, bool, error) {
	entry, ok := k.keys[name]
	if !ok {
		return nil, nil, false, fmt.Errorf("unknown CA %q", name)
	}
	return entry.certificate, entry.key, false, nil
}

func newKeystore(basePath string, cas []string) (pki.Keystore, error) {
	keystore := &keystore{
		keys: map[string]keystoreEntry{},
	}
	for _, name := range cas {
		certBytes, err := ioutil.ReadFile(path.Join(basePath, name+".pem"))
		if err != nil {
			return nil, fmt.Errorf("reading %q certificate: %v", name, err)
		}
		certificate, err := pki.ParsePEMCertificate(certBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing %q certificate: %v", name, err)
		}

		keyBytes, err := ioutil.ReadFile(path.Join(basePath, name+"-key.pem"))
		if err != nil {
			return nil, fmt.Errorf("reading %q key: %v", name, err)
		}
		key, err := pki.ParsePEMPrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing %q key: %v", name, err)
		}

		keystore.keys[name] = keystoreEntry{
			certificate: certificate,
			key:         key,
		}
	}

	return keystore, nil
}
