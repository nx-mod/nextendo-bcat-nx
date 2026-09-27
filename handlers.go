package main

// HTTP surface.
//
// The console reaches three hosts, routed here by sni-router:
//
//	bcat-list-*   the delivery-cache index for a title
//	bcat-data-*   a file's data (served as a signed container) by digest
//	bcat-topics-* topic metadata
//
// Requests are routed by the Host prefix; the title id and digest come from the
// path or a query parameter. The REAL Nintendo paths and the binary index
// layout are not fully published (see NOTES.md) — this serves a documented
// JSON index and digest-addressed containers, and also accepts explicit
// /list, /data and /topics paths so it can be driven without DNS.

import (
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type bcatServer struct {
	key *rsa.PrivateKey

	listReqs  atomic.Int64
	dataReqs  atomic.Int64
	dataBytes atomic.Int64
	topicReqs atomic.Int64
	misses    atomic.Int64
}

func (s *bcatServer) listen() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.route)

	addr := fmt.Sprintf(":%d", httpPort)
	srv := &http.Server{Addr: addr, Handler: logRequests(mux), ReadHeaderTimeout: 15 * time.Second}

	if certFile != "" && keyFile != "" {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS10}
		log.Printf("[BCAT] listening HTTPS %s (cert=%s)", addr, certFile)
		return srv.ListenAndServeTLS(certFile, keyFile)
	}
	log.Printf("[BCAT] listening HTTP %s (TLS terminated by sni-router)", addr)
	return srv.ListenAndServe()
}

// route dispatches by Host prefix, then by explicit path.
func (s *bcatServer) route(w http.ResponseWriter, r *http.Request) {
	host := strings.ToLower(r.Host)
	path := r.URL.Path
	switch {
	case strings.HasPrefix(host, "bcat-list") || strings.HasPrefix(path, "/list"):
		s.handleList(w, r)
	case strings.HasPrefix(host, "bcat-data") || strings.HasPrefix(path, "/data"):
		s.handleData(w, r)
	case strings.HasPrefix(host, "bcat-topics") || strings.HasPrefix(path, "/topics"):
		s.handleTopics(w, r)
	case path == "/healthz":
		fmt.Fprintln(w, "ok")
	default:
		s.misses.Add(1)
		http.NotFound(w, r)
	}
}

// titleFrom reads the title id from ?title= or the last hex path segment.
func titleFrom(r *http.Request) (uint64, bool) {
	if v := r.URL.Query().Get("title"); v != "" {
		if n, err := strconv.ParseUint(strings.TrimPrefix(v, "0x"), 16, 64); err == nil {
			return n, true
		}
	}
	for _, seg := range strings.Split(r.URL.Path, "/") {
		if len(seg) == 16 {
			if n, err := strconv.ParseUint(seg, 16, 64); err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

func (s *bcatServer) paramsFor(dc *deliveryCache) containerParams {
	pass := dc.Passphrase
	if pass == "" {
		pass = defaultPassphrase
	}
	return containerParams{
		TitleID:    dc.TitleID,
		Passphrase: pass,
		CryptoType: cryptoAES128CTR,
		HashType:   hashSHA256,
	}
}

// handleList returns the delivery-cache index for a title.
func (s *bcatServer) handleList(w http.ResponseWriter, r *http.Request) {
	s.listReqs.Add(1)
	title, ok := titleFrom(r)
	if !ok {
		http.Error(w, "no title id", http.StatusBadRequest)
		return
	}
	dc, err := scanTitle(contentDir, title)
	if err != nil || len(dc.Files) == 0 {
		// No content: an empty index, not an error — the console treats it as
		// "nothing new", exactly the safe answer.
		writeJSON(w, map[string]any{"titleId": titleHex(title), "directories": []string{}, "files": []cacheFile{}})
		return
	}
	writeJSON(w, map[string]any{
		"titleId":     titleHex(title),
		"directories": dc.directories(),
		"files":       dc.Files,
	})
	log.Printf("[BCAT] list %s -> %d file(s) in %d dir(s)", titleHex(title), len(dc.Files), len(dc.directories()))
}

// handleData serves one file, wrapped in a signed container, addressed by digest.
func (s *bcatServer) handleData(w http.ResponseWriter, r *http.Request) {
	s.dataReqs.Add(1)
	title, ok := titleFrom(r)
	if !ok {
		http.Error(w, "no title id", http.StatusBadRequest)
		return
	}
	digest := r.URL.Query().Get("digest")
	if digest == "" {
		// last path segment that looks like a 16-byte hex digest
		for _, seg := range strings.Split(r.URL.Path, "/") {
			if len(seg) == digestSize*2 {
				digest = seg
			}
		}
	}
	dc, err := scanTitle(contentDir, title)
	if err != nil {
		http.Error(w, "unknown title", http.StatusNotFound)
		return
	}
	f := dc.find(digest)
	if f == nil {
		s.misses.Add(1)
		http.NotFound(w, r)
		return
	}
	plain, err := readFile(f.path)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	container, err := buildContainer(s.paramsFor(dc), plain, s.key)
	if err != nil {
		http.Error(w, "container error", http.StatusInternalServerError)
		return
	}
	s.dataBytes.Add(int64(len(container)))
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(container)
	log.Printf("[BCAT] data %s %s/%s -> %d bytes (signed)", titleHex(title), f.Dir, f.Name, len(container))
}

// handleTopics answers the topic query with an empty topic set (no push topics).
func (s *bcatServer) handleTopics(w http.ResponseWriter, r *http.Request) {
	s.topicReqs.Add(1)
	title, _ := titleFrom(r)
	writeJSON(w, map[string]any{"titleId": titleHex(title), "topics": []string{}})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
