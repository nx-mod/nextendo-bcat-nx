package main

// News delivered the console's own way: the HOME menu's News fetches its channel catalog, the channel lists
// and each item from the BCAT hosts (bcat-topics / bcat-list / bcat-data), all redirected here. Formats are
// those of CrustySean's BCAT-Toolbox (based on Random06457's BCAT-Manager) and the console's own local news
// records; every reply is a BCAT content container (plaintext, signed with the local key, which a console
// with the nextendo_bcat_sig patch accepts).
//
// Items are one JSON file each in BCAT_NEWS_DIR, re-read on every request: see NEWS.md.
//
//	GET bcat-topics  /api/nx/v1/topics/catalog              the channels (News -> Find channels)
//	GET bcat-topics  /api/nx/v1/topics/<channel>/detail     one channel, with its latest items' URLs
//	GET bcat-topics  /api/nx/v1/topics/<channel>/icon       its icon
//	GET bcat-topics  /api/nx/v1/titles/<title>/topics       channels tied to a title (none)
//	GET bcat-list    /api/nx/v1/list/<channel>              a channel's items (nx_news, nx_notice, ...)
//	GET bcat-data    /api/nx/v1/news/<channel>/<news_id>    one item (the news record)

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	qlaunchTitle   = 0x0100000000001000 // News content is keyed to the HOME menu's title
	nextendoTopic  = "nx_news_nextendo" // the Nextendo channel, listed under Find channels
	newsDataHost   = "https://bcat-data-lp1.cdn.nintendo.net"
	defaultTopic   = "nx_news" // the main News feed: shown without subscribing
	newsImageLimit = 1 << 20
)

var newsDir = envOr("BCAT_NEWS_DIR", "news")

// newsFile is one item as written in BCAT_NEWS_DIR/<name>.json (see NEWS.md).
type newsFile struct {
	Title    string         `json:"title"`
	Body     string         `json:"body"`
	Footer   string         `json:"footer"`
	Channel  string         `json:"channel"`  // nx_news (default), nx_notice, or nextendo
	Date     string         `json:"date"`     // YYYY-MM-DD or RFC 3339; newest first
	Image    string         `json:"image"`    // file next to the JSON; default <name>.jpg if present
	Picture  bool           `json:"picture"`  // also show the image full-size in the body
	Button   *newsButton    `json:"button"`   // optional "more" button
	Movie    string         `json:"movie"`    // optional video URL in the body
	Priority int            `json:"priority"` // higher first (default 50)
	ID       uint32         `json:"id"`       // optional; default derived from the file name
	Extra    map[string]any `json:"extra"`    // raw fields added to the record as-is (advanced)

	name string // file name without .json
	when time.Time
	img  []byte
}

type newsButton struct {
	Type    string `json:"type"` // browser | game | shop | settings
	Text    string `json:"text"`
	URL     string `json:"url"`      // browser
	TitleID string `json:"title_id"` // game
	Query   string `json:"query"`    // shop
	Applet  int    `json:"applet"`   // settings: system applet type (2 parental controls, 4 news settings)
}

// loadNewsDir reads every item, newest first. Bad files are logged and skipped.
func loadNewsDir() []newsFile {
	var out []newsFile
	matches, _ := filepath.Glob(filepath.Join(newsDir, "*.json"))
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var n newsFile
		if err := json.Unmarshal(raw, &n); err != nil {
			log.Printf("[BCAT news] %s: %v (skipped)", filepath.Base(path), err)
			continue
		}
		n.name = strings.TrimSuffix(filepath.Base(path), ".json")
		if n.Title == "" {
			n.Title = n.name
		}
		switch n.Channel {
		case "", "news":
			n.Channel = defaultTopic
		case "notice":
			n.Channel = "nx_notice"
		case "nextendo":
			n.Channel = nextendoTopic
		}
		if n.Priority == 0 {
			n.Priority = 50
		}
		if n.ID == 0 {
			h := fnv.New32a()
			h.Write([]byte(n.name))
			n.ID = 700000000 + h.Sum32()%100000000
		}
		n.when = parseNewsDate(n.Date, path)
		imgName := n.Image
		if imgName == "" {
			imgName = n.name + ".jpg"
		}
		if b, err := os.ReadFile(filepath.Join(newsDir, imgName)); err == nil && len(b) <= newsImageLimit {
			n.img = b
		} else if n.Image != "" {
			log.Printf("[BCAT news] %s: image %s: %v", n.name, n.Image, err)
		}
		if n.img == nil {
			n.img = defaultNewsImage()
		}
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].when.After(out[j].when) })
	return out
}

func parseNewsDate(s, path string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	if st, err := os.Stat(path); err == nil {
		return st.ModTime()
	}
	return time.Now()
}

var defaultImage []byte

// defaultNewsImage is a plain Nextendo-red 256x256 JPEG, for items without an image.
func defaultNewsImage() []byte {
	if defaultImage == nil {
		m := image.NewRGBA(image.Rect(0, 0, 256, 256))
		draw.Draw(m, m.Bounds(), &image.Uniform{color.RGBA{230, 0, 18, 255}}, image.Point{}, draw.Src)
		var buf bytes.Buffer
		jpeg.Encode(&buf, m, &jpeg.Options{Quality: 90})
		defaultImage = buf.Bytes()
	}
	return defaultImage
}

func topicName(topic string) string {
	switch topic {
	case nextendoTopic:
		return "Nextendo"
	case "nx_notice":
		return "Nextendo Notices"
	}
	return "Nextendo Network"
}

// record is one item in the console's news record format (as the console's own local notices).
func (n newsFile) record() omap {
	r := omap{
		{"version", omap{{"format", 1}, {"semantics", 1}}},
		{"news_id", n.ID},
		{"published_at", n.when.Unix()},
		{"pickup_limit", 1209600},
		{"priority", n.Priority},
		{"deletion_priority", 100},
		{"language", "en-US"},
		{"supported_languages", []string{"en-US"}},
		{"display_type", "NORMAL"},
		{"topic_id", n.Channel},
		{"no_photography", 0},
		{"surprise", 0},
		{"essential_pickup_limit", 0},
		{"movie", 0},
		{"subject", omap{{"caption", 1}, {"text", n.Title}}},
		{"topic_name", topicName(n.Channel)},
		{"list_image", n.img},
	}
	if n.Footer != "" {
		r = append(r, kv{"footer", omap{{"text", n.Footer}}})
	}
	if b := n.Button; b != nil {
		var more omap
		switch b.Type {
		case "browser":
			more = omap{{"browser", omap{{"text", b.Text}, {"url", b.URL}}}}
		case "game":
			tid, _ := strconv.ParseUint(strings.TrimPrefix(b.TitleID, "0x"), 16, 64)
			more = omap{{"game", omap{{"application_ids", []any{tid}}, {"application_arg", []byte{}}, {"query", ""}, {"text", b.Text}}}}
		case "shop":
			more = omap{{"shop", omap{{"query", b.Query}, {"text", b.Text}}}}
		case "settings":
			applet := b.Applet
			if applet == 0 {
				applet = 4
			}
			more = omap{{"system_applet", omap{{"type", applet}, {"text", b.Text}}}}
		}
		if more != nil {
			r = append(r, kv{"more", more})
		}
	}
	body := omap{{"text", n.Body}}
	if n.Picture {
		body = append(body, kv{"main_image_height", 256}, kv{"main_image", n.img})
	}
	if n.Movie != "" {
		body = append(body, kv{"movie_url", n.Movie})
	}
	r = append(r, kv{"body", body})
	for k, v := range n.Extra {
		r = append(r, kv{k, v})
	}
	return r
}

func newsDataURL(n newsFile) string {
	return fmt.Sprintf("%s/api/nx/v1/news/%s/%d", newsDataHost, n.Channel, n.ID)
}

// News containers are encrypted with the HOME menu's News passphrase and a salt picked by the header's
// secret index (BCAT-Toolbox's DecryptBCAT). Those values are Nintendo's, so they are not in this repo:
// BCAT_NEWS_SECRETS names a local JSON file {"passphrase": "...", "salts": ["...", x32]}. Without it,
// containers are plaintext.
type newsSecretsFile struct {
	Passphrase string   `json:"passphrase"`
	Salts      []string `json:"salts"`
}

var newsSecrets = func() *newsSecretsFile {
	path := os.Getenv("BCAT_NEWS_SECRETS")
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[BCAT news] secrets %s: %v (containers stay plaintext)", path, err)
		return nil
	}
	var s newsSecretsFile
	if json.Unmarshal(raw, &s) != nil || s.Passphrase == "" || len(s.Salts) != 32 {
		log.Printf("[BCAT news] secrets %s: need a passphrase and 32 salts (containers stay plaintext)", path)
		return nil
	}
	log.Printf("[BCAT news] containers encrypted (AES-128-CTR)")
	return &s
}()

// container wraps a News payload the way the console expects it (HOME menu title, signed; encrypted when
// the News secrets are configured).
func (s *bcatServer) newsContainer(payload []byte) ([]byte, error) {
	p := containerParams{TitleID: qlaunchTitle, CryptoType: cryptoPlaintext, HashType: hashSHA256}
	if sec := newsSecrets; sec != nil {
		var b [1]byte
		rand.Read(b[:])
		idx := b[0] % 32
		p.CryptoType, p.Passphrase, p.SecretData, p.SecretIdx = cryptoAES128CTR, sec.Passphrase, sec.Salts[idx], idx
	}
	return buildContainer(p, payload, s.key)
}

func (s *bcatServer) writeNewsContainer(w http.ResponseWriter, payload []byte) {
	c, err := s.newsContainer(payload)
	if err != nil {
		http.Error(w, "container error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(c)
}

func channelEntry(topic string, items []newsFile) omap {
	last := int64(0)
	for _, n := range items {
		if n.Channel == topic && n.when.Unix() > last {
			last = n.when.Unix()
		}
	}
	return omap{
		{"topic_id", topic},
		{"name", topicName(topic)},
		{"publisher", "Nextendo Network"},
		{"description", "News from the Nextendo Network."},
		{"publishing_time", int64(1735689600)},
		{"last_posted_at", last},
		{"important", false},
	}
}

// handleNewsTopics answers the bcat-topics host.
func (s *bcatServer) handleNewsTopics(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	items := loadNewsDir()
	switch {
	case p == "/api/nx/v1/topics/catalog":
		s.writeNewsContainer(w, mpack([]omap{channelEntry(nextendoTopic, items)}))
		return true
	case strings.HasPrefix(p, "/api/nx/v1/topics/") && strings.HasSuffix(p, "/detail"):
		topic := strings.TrimSuffix(strings.TrimPrefix(p, "/api/nx/v1/topics/"), "/detail")
		d := channelEntry(topic, items)
		urls := []string{}
		for _, n := range items {
			if n.Channel == topic {
				urls = append(urls, newsDataURL(n))
			}
		}
		d = append(d, kv{"latest_news_urls", urls})
		s.writeNewsContainer(w, mpack(d))
		return true
	case strings.HasPrefix(p, "/api/nx/v2/topics/") && strings.HasSuffix(p, "/online_archives"):
		// A channel's older items (opening a channel asks after its detail). The format is not known: an empty
		// list, so the channel shows the detail's latest_news_urls.
		s.writeNewsContainer(w, mpack([]any{}))
		return true
	case strings.HasPrefix(p, "/api/nx/v1/topics/") && strings.HasSuffix(p, "/icon"):
		s.writeNewsContainer(w, defaultNewsImage())
		return true
	case strings.HasPrefix(p, "/api/nx/v1/titles/") && strings.HasSuffix(p, "/topics"):
		s.writeNewsContainer(w, mpack([]string{}))
		return true
	}
	return false
}

// handleNewsList answers /api/nx/v1/list/<channel> (not nx_data_*): the channel's items.
func (s *bcatServer) handleNewsList(w http.ResponseWriter, r *http.Request, topic string) {
	var data []any
	for _, n := range loadNewsDir() {
		if n.Channel != topic {
			continue
		}
		rec := mpack(n.record())
		c, _ := s.newsContainer(rec)
		data = append(data, omap{
			{"news_id", n.ID},
			{"version", omap{{"format", 1}, {"semantics", 1}}},
			{"default_language", "en-US"},
			{"publishing_time", n.when.Unix()},
			{"deletion_priority", 100},
			{"languages", []any{omap{
				{"language", "en-US"}, {"data_id", n.ID}, {"url", newsDataURL(n)}, {"size", len(c)}, {"overwrite", false},
			}}},
		})
	}
	if data == nil {
		data = []any{}
	}
	sum := sha256.Sum256(mpack(data))
	list := omap{
		{"topic_id", topic},
		{"service_status", "in_service"}, // bcat 22.5.0 accepts "in_service" or "expired" only (else 0x67d)
		{"na_required", false},
		{"test_distribution", false},
		{"directories", []any{omap{
			{"name", topic}, {"mode", "copy"}, {"digest", hex.EncodeToString(sum[:16])}, {"data_list", data},
		}}},
	}
	log.Printf("[BCAT news] list %s -> %d item(s)", topic, len(data))
	s.writeNewsContainer(w, mpack(list))
}

// handleNewsData answers /api/nx/v1/news/<channel>/<news_id>: one item.
func (s *bcatServer) handleNewsData(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/nx/v1/news/"), "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseUint(parts[1], 10, 32)
	for _, n := range loadNewsDir() {
		if n.Channel == parts[0] && uint64(n.ID) == id {
			s.writeNewsContainer(w, mpack(n.record()))
			return
		}
	}
	http.NotFound(w, r)
}
