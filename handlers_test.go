package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// withContent points contentDir at a temp tree with one title/dir/file and
// returns the title id and the file's bytes.
func withContent(t *testing.T) (uint64, []byte) {
	t.Helper()
	old := contentDir
	dir := t.TempDir()
	contentDir = dir
	t.Cleanup(func() { contentDir = old })

	title := uint64(0x010064800F66A000)
	data := []byte("advance-wars map-share payload \x00\x01\x02 binary")
	fdir := filepath.Join(dir, titleHex(title), "netmapshare")
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "map001.dat"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, titleHex(title), "passphrase"), []byte("the-passphrase"), 0o644); err != nil {
		t.Fatal(err)
	}
	return title, data
}

func TestListThenDataRoundTrip(t *testing.T) {
	title, data := withContent(t)
	s := &bcatServer{key: testKey(t)}

	// list
	rec := httptest.NewRecorder()
	s.handleList(rec, httptest.NewRequest("GET", "/list?title="+titleHex(title), nil))
	if rec.Code != 200 {
		t.Fatalf("list code %d", rec.Code)
	}
	var idx struct {
		Directories []string    `json:"directories"`
		Files       []cacheFile `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx.Files) != 1 || idx.Files[0].Dir != "netmapshare" || idx.Files[0].Name != "map001.dat" {
		t.Fatalf("index %+v", idx)
	}
	if len(idx.Directories) != 1 || idx.Directories[0] != "netmapshare" {
		t.Fatalf("dirs %v", idx.Directories)
	}
	digest := idx.Files[0].Digest

	// data by digest
	rec = httptest.NewRecorder()
	s.handleData(rec, httptest.NewRequest("GET", "/data?title="+titleHex(title)+"&digest="+digest, nil))
	if rec.Code != 200 {
		t.Fatalf("data code %d", rec.Code)
	}
	blob := rec.Body.Bytes()
	if string(blob[0:4]) != containerMagic {
		t.Fatalf("data is not a container: %x", blob[0:4])
	}
	// the console (with our public key) verifies + decrypts back to the file
	dc, _ := scanTitle(contentDir, title)
	plain, err := verifyContainer(s.paramsFor(dc), blob, &s.key.PublicKey)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !bytes.Equal(plain, data) {
		t.Fatalf("decrypted %q, want %q", plain, data)
	}
}

func TestListEmptyForUnknownTitle(t *testing.T) {
	withContent(t)
	s := &bcatServer{key: testKey(t)}
	rec := httptest.NewRecorder()
	s.handleList(rec, httptest.NewRequest("GET", "/list?title=0000000000000001", nil))
	if rec.Code != 200 {
		t.Fatalf("code %d (empty index should be a 200, not an error)", rec.Code)
	}
}

func TestDataMissDigest404(t *testing.T) {
	title, _ := withContent(t)
	s := &bcatServer{key: testKey(t)}
	rec := httptest.NewRecorder()
	s.handleData(rec, httptest.NewRequest("GET", "/data?title="+titleHex(title)+"&digest=00000000000000000000000000000000", nil))
	if rec.Code != 404 {
		t.Fatalf("missing digest code %d, want 404", rec.Code)
	}
}
