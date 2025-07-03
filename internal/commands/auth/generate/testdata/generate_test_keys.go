package main

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

func main() {
	// Create testdata directory if it doesn't exist
	testdataDir := "testdata"
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		log.Fatalf("Failed to create testdata directory: %v", err)
	}

	// Generate Ed25519 key
	path := filepath.Join(testdataDir, "ed25519_key.pem")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := generateEd25519Key(path); err != nil {
			log.Fatalf("Failed to generate Ed25519 key: %v", err)
		}
	}

	// Generate ECDSA keys for different curves
	curves := []struct {
		name  string
		curve elliptic.Curve
	}{
		{"ecdsa_p256", elliptic.P256()},
		{"ecdsa_p384", elliptic.P384()},
		{"ecdsa_p521", elliptic.P521()},
	}

	for _, c := range curves {
		path := filepath.Join(testdataDir, c.name+"_key.pem")
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := generateEcdsaKey(path, c.curve); err != nil {
				log.Fatalf("Failed to generate ECDSA key for %s: %v", c.name, err)
			}
		}
	}

	// Generate P-224 key separately since it's not supported by SSH
	path = filepath.Join(testdataDir, "ecdsa_p224_key.pem")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := generateEcdsaKeyManual(path, elliptic.P224()); err != nil {
			log.Fatalf("Failed to generate ECDSA P-224 key: %v", err)
		}
	}

	// Generate RSA keys with different bit sizes
	rsaSizes := []struct {
		name string
		bits int
	}{
		{"rsa_2047", 2047},
		{"rsa_8193", 8193},
		{"rsa_2048", 2048},
		{"rsa_8192", 8192},
	}

	for _, r := range rsaSizes {
		path := filepath.Join(testdataDir, r.name+"_key.pem")
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := generateRsaKey(path, r.bits); err != nil {
				log.Fatalf("Failed to generate RSA key for %s: %v", r.name, err)
			}
		}
	}
}

func generateEd25519Key(filename string) error {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	return savePrivateKeyAsSSH(filename, privateKey)
}

func generateEcdsaKey(filename string, curve elliptic.Curve) error {
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return err
	}

	return savePrivateKeyAsSSH(filename, privateKey)
}

func generateEcdsaKeyManual(filename string, curve elliptic.Curve) error {
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return err
	}

	return savePrivateKeyManual(filename, privateKey)
}

func generateRsaKey(filename string, bits int) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}

	return savePrivateKeyAsSSH(filename, privateKey)
}

func savePrivateKeyAsSSH(filename string, privateKey crypto.PrivateKey) error {
	// Convert to SSH format
	sshPrivateKey, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return err
	}

	// Write to file
	encoded := &bytes.Buffer{}
	err = pem.Encode(encoded, sshPrivateKey)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, encoded.Bytes(), 0o644)
}

func savePrivateKeyManual(filename string, privateKey crypto.PrivateKey) error {
	var keyBytes []byte
	var keyType string
	var err error

	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
		keyType = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		keyBytes, err = x509.MarshalECPrivateKey(key)
		keyType = "EC PRIVATE KEY"
	case ed25519.PrivateKey:
		keyBytes, err = x509.MarshalPKCS8PrivateKey(key)
		keyType = "PRIVATE KEY"
	default:
		return fmt.Errorf("unsupported key type: %T", privateKey)
	}

	if err != nil {
		return err
	}

	pemBlock := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}

	encoded := &bytes.Buffer{}
	err = pem.Encode(encoded, pemBlock)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, encoded.Bytes(), 0o644)
}
