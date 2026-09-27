package main

// News: a small example of pushing content to the console through BCAT topics.
//
// The Switch News applet pulls topic metadata + content from the BCAT topics
// host (bcat-topics-lp1.cdn.nintendo.net), which the stack redirects here. This
// serves a configurable list of news items — by default a single "nextendo news
// test" headline — as topic metadata and as a signed BCAT container, so the
// same item can be delivered either way.
//
// NOTE: the News applet's exact on-console container schema (thumbnail, per-
// topic signing, layout) is internal and not fully published; this serves the
// tractable JSON + a signed container and logs the requests, the same capture-
// surface approach the rest of the stack uses. See NOTES.md.

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// newsItem is one entry in the Nextendo news feed.
type newsItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Time    int64  `json:"time"`
}

var (
	newsMu   sync.RWMutex
	newsFeed []newsItem
)

// loadNews reads news.json if present, else seeds the test item. Safe to call at
// startup; hot-reloadable by calling again.
func loadNews(path string) {
	items := []newsItem{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &items); err != nil {
			log.Printf("[BCAT news] %s: %v (using default)", path, err)
			items = nil
		}
	}
	if len(items) == 0 {
		items = []newsItem{{
			ID: "nextendo-test", Title: "Nextendo",
			Message: "nextendo news test", Time: time.Now().Unix(),
		}}
	}
	for i := range items {
		if items[i].Time == 0 {
			items[i].Time = time.Now().Unix()
		}
	}
	newsMu.Lock()
	newsFeed = items
	newsMu.Unlock()
	log.Printf("[BCAT news] loaded %d item(s) from %s", len(items), path)
}

func currentNews() []newsItem {
	newsMu.RLock()
	defer newsMu.RUnlock()
	out := make([]newsItem, len(newsFeed))
	copy(out, newsFeed)
	return out
}

// handleNews serves the news feed as JSON (a convenience endpoint for testing
// and for a simple client), at /news.
func (s *bcatServer) handleNews(w http.ResponseWriter, r *http.Request) {
	items := currentNews()
	writeJSON(w, map[string]any{"count": len(items), "news": items})
	if len(items) > 0 {
		log.Printf("[BCAT news] served %d item(s), first=%q", len(items), items[0].Message)
	}
}
