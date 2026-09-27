package main

// Monitoring for nextendo-dashboard: GET /api/stats?key=<DASH_TOKEN>, plus
// /healthz. Same idea as the sibling servers: request counters and the served
// titles, so BCAT shows on the shared dashboard.

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

var dashStart = time.Now()

type apiStats struct {
	ServerTime    string   `json:"serverTime"`
	UptimeSeconds int      `json:"uptimeSeconds"`
	ListRequests  int64    `json:"listRequests"`
	DataRequests  int64    `json:"dataRequests"`
	DataBytes     int64    `json:"dataBytes"`
	TopicRequests int64    `json:"topicRequests"`
	Misses        int64    `json:"misses"`
	Titles        []string `json:"titles"`
	Stack         string   `json:"stack"`
}

// servedTitles lists the title-id directories present under the content dir.
func servedTitles() []string {
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && len(e.Name()) == 16 {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func startDashboard(s *bcatServer) {
	authed := func(w http.ResponseWriter, r *http.Request) bool {
		if dashToken != "" && subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("key")), []byte(dashToken)) == 1 {
			return true
		}
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if !authed(w, r) {
			return
		}
		st := apiStats{
			ServerTime:    time.Now().Format("15:04:05"),
			UptimeSeconds: int(time.Since(dashStart).Seconds()),
			ListRequests:  s.listReqs.Load(),
			DataRequests:  s.dataReqs.Load(),
			DataBytes:     s.dataBytes.Load(),
			TopicRequests: s.topicReqs.Load(),
			Misses:        s.misses.Load(),
			Titles:        servedTitles(),
			Stack:         "bcat",
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(st)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok\n")) })

	log.Printf("[BCAT Dashboard] stats API on :%s (token=%v)", dashPort, dashToken != "")
	if err := http.ListenAndServe(":"+dashPort, mux); err != nil {
		log.Printf("[BCAT Dashboard] HTTP error: %v", err)
	}
}
