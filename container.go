package main

// BCAT content-container format.
//
// A BCAT delivery-cache data file is wrapped in a 0x120-byte header, documented
// on switchbrew (BCAT Content Container):
//
//	0x00 [4]    magic "bcat"
//	0x04 u8     unknown
//	0x05 u8     crypto type   1=AES-128-CTR 2=AES-192-CTR 3=AES-256-CTR else=plaintext
//	0x06 u8     hash type     0,2=SHA-1  1,3=SHA-256  (>3 = error)
//	0x07 u8     secret-data index (selects the secretdata string for the salt)
//	0x08 u64    reserved (0)
//	0x10 [0x10] base IV/CTR
//	0x20 [0x100] RSA-2048 signature over the container
//	0x120 ...   payload (AES-CTR encrypted when crypto type is set)
//
// The AES key is derived with PBKDF2-HMAC-SHA256, 4096 iterations, from the
// title's passphrase and a salt of fmt "%016x%s" over (title id, secretdata).
//
// The RSA-2048 signature is the piece a stock console verifies with Nintendo's
// public key. This project's on-console module replaces that key with a local
// one (see keys.go), so a container this server signs with the matching local
// PRIVATE key is accepted. The exact signed region and padding are Nintendo's
// and are not fully published; this uses RSA-PSS over the header (minus the
// signature) plus the payload, which both sides here agree on — CONFIRM against
// a console before relying on stock verification (see NOTES.md).

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
)

const (
	containerMagic = "bcat"
	headerSize     = 0x120
	sigOffset      = 0x20
	sigSize        = 0x100 // RSA-2048
	ivOffset       = 0x10
	ivSize         = 0x10
	pbkdf2Iters    = 4096
	passphraseMax  = 0x40
)

// crypto types (header[5]).
const (
	cryptoPlaintext = 0
	cryptoAES128CTR = 1
	cryptoAES192CTR = 2
	cryptoAES256CTR = 3
)

// hash types (header[6]).
const (
	hashSHA1a   = 0
	hashSHA256  = 1
	hashSHA1b   = 2
	hashSHA256b = 3
)

// containerParams describes how to build/verify one title's containers.
type containerParams struct {
	TitleID    uint64
	Passphrase string // the title's BCAT passphrase (<= 0x40 bytes)
	SecretData string // secretdata string for the salt (selected by index)
	SecretIdx  byte
	CryptoType byte // one of the cryptoAES* / cryptoPlaintext
	HashType   byte // one of the hash* constants
}

// deriveKey returns the AES key for a title, sized to the crypto type.
func (p containerParams) deriveKey() ([]byte, error) {
	keyLen := p.keyLen()
	if keyLen == 0 {
		return nil, nil // plaintext
	}
	pass := p.Passphrase
	if len(pass) > passphraseMax {
		pass = pass[:passphraseMax]
	}
	salt := fmt.Sprintf("%016x%s", p.TitleID, p.SecretData)
	return pbkdf2.Key(sha256.New, pass, []byte(salt), pbkdf2Iters, keyLen)
}

func (p containerParams) keyLen() int {
	switch p.CryptoType {
	case cryptoAES128CTR:
		return 16
	case cryptoAES192CTR:
		return 24
	case cryptoAES256CTR:
		return 32
	default:
		return 0
	}
}

func (p containerParams) newHash() hash.Hash {
	switch p.HashType {
	case hashSHA256, hashSHA256b:
		return sha256.New()
	default:
		return sha1.New()
	}
}

func (p containerParams) cryptoHash() crypto.Hash {
	switch p.HashType {
	case hashSHA256, hashSHA256b:
		return crypto.SHA256
	default:
		return crypto.SHA1
	}
}

// buildContainer wraps plaintext in a signed (and, per crypto type, encrypted)
// BCAT container. key is the local RSA-2048 private key that stands in for
// Nintendo's.
func buildContainer(p containerParams, plaintext []byte, key *rsa.PrivateKey) ([]byte, error) {
	if key.N.BitLen() != 2048 {
		return nil, fmt.Errorf("container signing key must be RSA-2048, got %d bits", key.N.BitLen())
	}
	header := make([]byte, headerSize)
	copy(header[0:4], containerMagic)
	header[5] = p.CryptoType
	header[6] = p.HashType
	header[7] = p.SecretIdx

	iv := make([]byte, ivSize)
	payload := plaintext
	if p.keyLen() > 0 {
		aesKey, err := p.deriveKey()
		if err != nil {
			return nil, err
		}
		if _, err := rand.Read(iv); err != nil {
			return nil, err
		}
		payload = ctrCrypt(aesKey, iv, plaintext)
	}
	copy(header[ivOffset:ivOffset+ivSize], iv)

	// Sign the header (minus the signature region) plus the payload.
	sig, err := signContainer(p, header, payload, key)
	if err != nil {
		return nil, err
	}
	copy(header[sigOffset:sigOffset+sigSize], sig)

	return append(header, payload...), nil
}

// signContainer produces the RSA-PSS signature over the signed region.
func signContainer(p containerParams, header, payload []byte, key *rsa.PrivateKey) ([]byte, error) {
	h := p.newHash()
	h.Write(signedRegion(header, payload))
	digest := h.Sum(nil)
	return rsa.SignPSS(rand.Reader, key, p.cryptoHash(), digest, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

// signedRegion is the bytes the signature covers: the header up to the
// signature field, then the header after it, then the payload.
func signedRegion(header, payload []byte) []byte {
	var b bytes.Buffer
	b.Write(header[:sigOffset])
	b.Write(header[sigOffset+sigSize:])
	b.Write(payload)
	return b.Bytes()
}

// parsedContainer is a decoded container.
type parsedContainer struct {
	CryptoType byte
	HashType   byte
	SecretIdx  byte
	IV         []byte
	Signature  []byte
	Payload    []byte // still encrypted when CryptoType != plaintext
}

var errBadContainer = errors.New("not a BCAT container")

// parseContainer splits a container into its header fields and payload.
func parseContainer(data []byte) (*parsedContainer, error) {
	if len(data) < headerSize || string(data[0:4]) != containerMagic {
		return nil, errBadContainer
	}
	return &parsedContainer{
		CryptoType: data[5],
		HashType:   data[6],
		SecretIdx:  data[7],
		IV:         append([]byte(nil), data[ivOffset:ivOffset+ivSize]...),
		Signature:  append([]byte(nil), data[sigOffset:sigOffset+sigSize]...),
		Payload:    append([]byte(nil), data[headerSize:]...),
	}, nil
}

// verifyContainer checks the signature against pub (the local public key that
// the module installs in place of Nintendo's) and returns the decrypted plaintext.
func verifyContainer(p containerParams, data []byte, pub *rsa.PublicKey) ([]byte, error) {
	pc, err := parseContainer(data)
	if err != nil {
		return nil, err
	}
	header := make([]byte, headerSize)
	copy(header, data[:headerSize])
	h := p.newHash()
	h.Write(signedRegion(header, pc.Payload))
	digest := h.Sum(nil)
	if err := rsa.VerifyPSS(pub, p.cryptoHash(), digest, pc.Signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}); err != nil {
		return nil, fmt.Errorf("container signature invalid: %w", err)
	}
	if p.keyLen() == 0 {
		return pc.Payload, nil
	}
	aesKey, err := p.deriveKey()
	if err != nil {
		return nil, err
	}
	return ctrCrypt(aesKey, pc.IV, pc.Payload), nil
}

// ctrCrypt runs AES-CTR (its own inverse) over data with key and iv.
func ctrCrypt(key, iv, data []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	out := make([]byte, len(data))
	cipher.NewCTR(block, iv).XORKeyStream(out, data)
	return out
}
