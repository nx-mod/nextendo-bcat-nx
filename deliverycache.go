package main

// Delivery cache: a title's BCAT content, taken from disk.
//
// Layout under BCAT_CONTENT:
//
//	<content>/<titleid-hex>/<directory>/<file>
//	<content>/<titleid-hex>/passphrase   (optional, one line: the title passphrase)
//
// nn::bcat models a title's delivery cache as a set of directories, each holding
// files; every file has a name (<=0x20), a size and a 0x10 digest. The console
// pulls an index (the list) of that metadata, then fetches each file's data by
// its digest and checks it. This builds that model from the files on disk; each
// data file is served wrapped in a signed container (container.go).
//
// UNCONFIRMED against a console: the exact digest algorithm and the on-wire
// index/list byte layout are Nintendo's and not fully published. This uses
// SHA-256[:16] as the file digest and a documented JSON+binary index; the digest
// only has to agree between the index and the data naming here, but a stock
// console recomputes it its own way — confirm from a capture (see NOTES.md).

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	dirNameMax  = 0x20
	fileNameMax = 0x20
	digestSize  = 0x10
)

// cacheFile is one file in a title's delivery cache.
type cacheFile struct {
	Dir    string `json:"directory"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"` // hex, 16 bytes
	path   string
}

// deliveryCache is one title's content.
type deliveryCache struct {
	TitleID    uint64      `json:"titleId"`
	Passphrase string      `json:"-"`
	Files      []cacheFile `json:"files"`
}

// scanTitle reads <content>/<titleid>/ into a deliveryCache. Names longer than
// the field width are skipped (the console cannot address them).
func scanTitle(contentDir string, titleID uint64) (*deliveryCache, error) {
	root := filepath.Join(contentDir, titleHex(titleID))
	dc := &deliveryCache{TitleID: titleID}

	if pass, err := os.ReadFile(filepath.Join(root, "passphrase")); err == nil {
		dc.Passphrase = strings.TrimSpace(string(pass))
	}

	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if !d.IsDir() || len(d.Name()) > dirNameMax {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, d.Name()))
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || len(f.Name()) > fileNameMax {
				continue
			}
			p := filepath.Join(root, d.Name(), f.Name())
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			dc.Files = append(dc.Files, cacheFile{
				Dir:    d.Name(),
				Name:   f.Name(),
				Size:   int64(len(data)),
				Digest: fileDigest(data),
				path:   p,
			})
		}
	}
	sort.Slice(dc.Files, func(i, j int) bool {
		if dc.Files[i].Dir != dc.Files[j].Dir {
			return dc.Files[i].Dir < dc.Files[j].Dir
		}
		return dc.Files[i].Name < dc.Files[j].Name
	})
	return dc, nil
}

// fileDigest returns the 16-byte file digest as hex. UNCONFIRMED algorithm
// (SHA-256[:16]); see the file header.
func fileDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return toHex(sum[:digestSize])
}

// find returns the file with the given digest, or nil.
func (dc *deliveryCache) find(digest string) *cacheFile {
	for i := range dc.Files {
		if strings.EqualFold(dc.Files[i].Digest, digest) {
			return &dc.Files[i]
		}
	}
	return nil
}

// directories returns the distinct directory names, in order.
func (dc *deliveryCache) directories() []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range dc.Files {
		if !seen[f.Dir] {
			seen[f.Dir] = true
			out = append(out, f.Dir)
		}
	}
	return out
}

func titleHex(titleID uint64) string {
	const hexdigits = "0123456789abcdef"
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = hexdigits[titleID&0xf]
		titleID >>= 4
	}
	return string(b[:])
}

func readFile(path string) ([]byte, error) { return os.ReadFile(path) }

func toHex(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, c := range b {
		out[i*2] = hexdigits[c>>4]
		out[i*2+1] = hexdigits[c&0xf]
	}
	return string(out)
}
