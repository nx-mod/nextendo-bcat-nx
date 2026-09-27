package main

// The local BCAT signing key — the replacement for Nintendo's.
//
// A stock console verifies every delivery-cache container against Nintendo's
// RSA-2048 public key (baked into nn::bcat). This project's on-console module
// (bcat-mitm) replaces that public key with the one below, so containers this
// server signs with the matching PRIVATE key are accepted. The private key
// never leaves the server; the public key is what goes into the module.
//
// The key is loaded from BCAT_KEY_FILE (PEM, PKCS#1). If the file is absent one
// is generated and written there on first run, and the matching public key is
// written next to it as <key>.pub.pem and .der, ready to embed in the module.

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"
)

// loadOrCreateKey returns the local RSA-2048 signing key, generating and
// persisting one if path does not exist.
func loadOrCreateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		key, perr := parsePrivateKeyPEM(raw)
		if perr != nil {
			return nil, fmt.Errorf("%s: %w", path, perr)
		}
		if key.N.BitLen() != 2048 {
			return nil, fmt.Errorf("%s: key is RSA-%d, BCAT needs RSA-2048", path, key.N.BitLen())
		}
		log.Printf("[BCAT] loaded signing key %s", path)
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	log.Printf("[BCAT] %s absent: generating a new RSA-2048 signing key", path)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	if err := writeKeyFiles(path, key); err != nil {
		// A generated key that cannot be persisted still works for this run, but
		// the module would need re-flashing next restart; warn loudly.
		log.Printf("[BCAT] WARNING: could not persist key: %v", err)
	}
	return key, nil
}

func parsePrivateKeyPEM(raw []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("no PEM block")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA key")
	}
	return rk, nil
}

// writeKeyFiles writes the private key (PKCS#1 PEM) and, beside it, the public
// key as PEM and DER for the module to embed.
func writeKeyFiles(path string, key *rsa.PrivateKey) error {
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(path, privPEM, 0o600); err != nil {
		return err
	}
	base := strings.TrimSuffix(path, ".pem")
	pubDER := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubDER})
	if err := os.WriteFile(base+".pub.pem", pubPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(base+".pub.der", pubDER, 0o644); err != nil {
		return err
	}
	log.Printf("[BCAT] wrote public key %s.pub.{pem,der} — embed the .der modulus in bcat-mitm", base)
	return nil
}

// publicModulus returns the raw 256-byte big-endian RSA modulus, the form the
// on-console module patches in over Nintendo's.
func publicModulus(key *rsa.PrivateKey) []byte {
	return key.N.FillBytes(make([]byte, 256))
}
