package main

import (
	"bytes"
	"crypto/rsa"
	"testing"
)

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := loadOrCreateKey(t.TempDir() + "/k.pem")
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestContainerRoundTripEncrypted(t *testing.T) {
	key := testKey(t)
	p := containerParams{TitleID: 0x010064800F66A000, Passphrase: "secret", CryptoType: cryptoAES128CTR, HashType: hashSHA256}
	plain := []byte("delivery cache file contents, whatever the game reads")

	blob, err := buildContainer(p, plain, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob[0:4]) != containerMagic {
		t.Fatalf("bad magic %x", blob[0:4])
	}
	if len(blob) != headerSize+len(plain) {
		t.Fatalf("length %d, want %d", len(blob), headerSize+len(plain))
	}
	// the payload must be encrypted (not equal to plaintext on the wire)
	if bytes.Equal(blob[headerSize:], plain) {
		t.Fatal("payload not encrypted")
	}
	// verify + decrypt with the matching public key (what the module installs)
	got, err := verifyContainer(p, blob, &key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestContainerPlaintext(t *testing.T) {
	key := testKey(t)
	p := containerParams{TitleID: 1, CryptoType: cryptoPlaintext, HashType: hashSHA1a}
	plain := []byte("plain")
	blob, err := buildContainer(p, plain, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(blob[headerSize:], plain) {
		t.Fatal("plaintext payload should be verbatim")
	}
	got, err := verifyContainer(p, blob, &key.PublicKey)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("verify plaintext: %v %q", err, got)
	}
}

func TestContainerRejectsWrongKey(t *testing.T) {
	key := testKey(t)
	other := testKey(t)
	p := containerParams{TitleID: 1, Passphrase: "x", CryptoType: cryptoAES128CTR, HashType: hashSHA256}
	blob, _ := buildContainer(p, []byte("data"), key)
	if _, err := verifyContainer(p, blob, &other.PublicKey); err == nil {
		t.Fatal("a container signed by one key verified under another")
	}
}

func TestContainerRejectsTamper(t *testing.T) {
	key := testKey(t)
	p := containerParams{TitleID: 1, Passphrase: "x", CryptoType: cryptoAES128CTR, HashType: hashSHA256}
	blob, _ := buildContainer(p, []byte("data payload here"), key)
	blob[headerSize+2] ^= 0xff // flip a payload byte
	if _, err := verifyContainer(p, blob, &key.PublicKey); err == nil {
		t.Fatal("tampered payload verified")
	}
}

func TestDeriveKeyDeterministicAndSized(t *testing.T) {
	p := containerParams{TitleID: 0xABCD, Passphrase: "pw", SecretData: "sd", CryptoType: cryptoAES256CTR}
	a, err := p.deriveKey()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := p.deriveKey()
	if !bytes.Equal(a, b) {
		t.Fatal("key derivation not deterministic")
	}
	if len(a) != 32 {
		t.Fatalf("AES-256 key len %d", len(a))
	}
	// a different title id changes the key (salt includes the title)
	p.TitleID = 0x1234
	c, _ := p.deriveKey()
	if bytes.Equal(a, c) {
		t.Fatal("title id not mixed into the key")
	}
}
