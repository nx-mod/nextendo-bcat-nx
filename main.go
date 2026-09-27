// Command nextendo-bcat-nx serves Nintendo Switch BCAT (the "d4c" delivery-cache
// service) for Nextendo Network, so a title's background content comes from the
// stack instead of Nintendo's CDN.
//
// A console fetches its delivery cache over HTTPS from bcat-list / bcat-topics /
// bcat-data (*.cdn.nintendo.net). sni-router sends those hosts here (TLS
// passthrough, like the game servers); this server answers them, wrapping each
// file in a signed BCAT container (container.go). A stock console verifies the
// container against Nintendo's key, so the companion on-console module
// (bcat-mitm) replaces that key with this server's local one (keys.go).
//
// This closes the gap the game notes keep hitting: Borderlands' "unable to load
// data" and Advance Wars' bcat-list/bcat-topics both fall through to the real
// Nintendo CDN today.
//
//	server            run the server
//	server pubkey     print the local public key (modulus) for the module and exit
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

var (
	nextendoHost = envOr("NEXTENDO_HOST", "127.0.0.1")
	httpPort     = envOrInt("BCAT_PORT", 8470)
	certFile     = envOr("CERT_FILE", "")
	keyFile      = envOr("KEY_FILE", "")

	contentDir = envOr("BCAT_CONTENT", "content")
	keyPath    = envOr("BCAT_KEY_FILE", "bcat_signing_key.pem")

	// Default per-title BCAT passphrase, when a title has no passphrase file.
	// The real passphrase is title-specific; container verification only needs
	// the server and console to agree, which the module ensures.
	defaultPassphrase = envOr("BCAT_PASSPHRASE", "")

	dashPort  = envOr("DASH_PORT", "8099")
	dashToken = envOr("DASH_TOKEN", "")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

var newsPath = envOr("BCAT_NEWS", "news.json")

func main() {
	log.SetOutput(os.Stdout)

	loadNews(newsPath)

	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		log.Fatalf("[BCAT] signing key: %v", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "pubkey" {
		mod := publicModulus(key)
		fmt.Printf("RSA-2048 public modulus (embed in bcat-mitm, big-endian, %d bytes):\n%s\n", len(mod), toHex(mod))
		return
	}

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		log.Printf("[BCAT] content dir %s: %v", contentDir, err)
	}

	srv := &bcatServer{key: key}
	go startDashboard(srv)

	log.Printf("[BCAT] content=%s key=%s (public modulus via `server pubkey`)", contentDir, keyPath)
	if err := srv.listen(); err != nil {
		log.Fatalf("[BCAT] stopped: %v", err)
	}
}
