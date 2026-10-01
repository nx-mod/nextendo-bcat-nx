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

// newsChannel is a custom channel, defined in BCAT_NEWS_DIR/channels.json (see NEWS.md).
type newsChannel struct {
	Name        string   `json:"name"`        // short name used in news files ("channel": "<name>")
	Topic       string   `json:"topic"`       // topic id; default nx_news_<name>
	Title       string   `json:"title"`       // shown as the channel's name
	Description string   `json:"description"` // shown on the channel's page
	Publisher   string   `json:"publisher"`   // default newsPublisher
	Games       []string `json:"games"`       // title ids (16 hex digits) whose channel this is
	Default     bool     `json:"default"`     // every console follows it (the HOME menu's list)
}

// builtinChannels are always there: the Nextendo channel, followed by every console.
var builtinChannels = []newsChannel{{
	Name: "nextendo", Topic: nextendoTopic, Title: "Nextendo",
	Description: "News from the Nextendo Network.", Default: true,
}}

// newsChannels is the built-in channels then channels.json's, re-read on every request. A channel there with
// a built-in's name or topic replaces it.
func newsChannels() []newsChannel {
	out := append([]newsChannel{}, builtinChannels...)
	raw, err := os.ReadFile(filepath.Join(newsDir, "channels.json"))
	if err != nil {
		return out
	}
	var defs []newsChannel
	if err := json.Unmarshal(raw, &defs); err != nil {
		log.Printf("[BCAT news] channels.json: %v (ignored)", err)
		return out
	}
	for _, c := range defs {
		c.Name = strings.ToLower(strings.TrimSpace(c.Name))
		if c.Topic == "" && c.Name != "" {
			c.Topic = "nx_news_" + c.Name
		}
		if c.Topic == "" || c.Topic == defaultTopic || c.Topic == "nx_notice" {
			continue
		}
		if c.Title == "" {
			c.Title = c.Name
		}
		for i := range c.Games {
			c.Games[i] = strings.ToLower(strings.TrimPrefix(c.Games[i], "0x"))
		}
		replaced := false
		for i := range out {
			if out[i].Topic == c.Topic || (c.Name != "" && out[i].Name == c.Name) {
				out[i], replaced = c, true
			}
		}
		if !replaced {
			out = append(out, c)
		}
	}
	return out
}

// channelByTopic is the custom channel with that topic id, nil for Nintendo's own or an unknown one.
func channelByTopic(topic string) *newsChannel {
	for _, c := range newsChannels() {
		if c.Topic == topic {
			return &c
		}
	}
	return nil
}

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
	Related  []string       `json:"related"`  // other channels listed under "Related channels" in the opened item
	Featured bool           `json:"featured"` // on the lock screen and the featured row (the newest 3 only)
	Priority int            `json:"priority"` // advanced: 1000+ is featured, highest first; default 1500 if featured, else 100
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

// channelTopic is a news file's channel name as a topic id: news (or nothing) and notice are Nintendo's, then
// the custom channels' short names (nextendo, and channels.json's); anything else is taken as the topic id.
func channelTopic(c string) string {
	switch c {
	case "", "news":
		return defaultTopic
	case "notice":
		return "nx_notice"
	}
	for _, ch := range newsChannels() {
		if ch.Name == strings.ToLower(c) {
			return ch.Topic
		}
	}
	return c
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
		n.Channel = channelTopic(n.Channel)
		for i, c := range n.Related {
			n.Related[i] = channelTopic(c)
		}
		if n.Priority == 0 {
			n.Priority = normalPriority
			if n.Featured {
				n.Priority = featuredPriority
			}
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
	// The lock screen has 3 featured slots: only the newest maxFeatured stay featured.
	featured := 0
	for i := range out {
		if out[i].Priority < featuredFloor {
			continue
		}
		if featured++; featured > maxFeatured {
			out[i].Priority = normalPriority
		}
	}
	return out
}

// qlaunch (22.5.0) features items of priority 1000 and up, highest first, on the lock screen and the featured
// row; lower ones (above 50) are the ordinary "latest" list.
const (
	featuredFloor    = 1000
	featuredPriority = 1500
	normalPriority   = 100
	maxFeatured      = 3
)

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

// channelIconSize is the channel icon's side: qlaunch (22.5.0) decodes it into a 70x70 texture and shows a
// question mark for any other size.
const channelIconSize = 70

// channelIcon is the channel's icon, a 70x70 JPEG: BCAT_NEWS_DIR/<channel>-icon.jpg or icon.jpg if there is
// one, else default.jpg; scaled to size (area average).
func channelIcon(topic string) []byte {
	src := defaultNewsImage()
	for _, name := range []string{topic + "-icon.jpg", "icon.jpg"} {
		if b, err := os.ReadFile(filepath.Join(newsDir, name)); err == nil && len(b) > 0 {
			src = b
			break
		}
	}
	return fitJPEG(src, channelIconSize, channelIconSize)
}

// fitJPEG returns src as a w x h JPEG: scaled to fit (area average), centred, padded with its corner colour.
// qlaunch decodes each image into a texture of a fixed size and rejects any other.
func fitJPEG(src []byte, w, h int) []byte {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return src
	}
	sb := img.Bounds()
	if sb.Dx() == w && sb.Dy() == h {
		return src
	}
	// The scaled size that fits inside w x h.
	dw, dh := w, sb.Dy()*w/sb.Dx()
	if dh > h {
		dw, dh = sb.Dx()*h/sb.Dy(), h
	}
	dw, dh = max(dw, 1), max(dh, 1)
	ox, oy := (w-dw)/2, (h-dh)/2
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	cr, cg, cb, _ := img.At(sb.Min.X, sb.Min.Y).RGBA()
	draw.Draw(out, out.Bounds(), &image.Uniform{color.RGBA{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), 255}}, image.Point{}, draw.Src)
	for y := 0; y < dh; y++ {
		y0, y1 := sb.Min.Y+y*sb.Dy()/dh, sb.Min.Y+(y+1)*sb.Dy()/dh
		for x := 0; x < dw; x++ {
			x0, x1 := sb.Min.X+x*sb.Dx()/dw, sb.Min.X+(x+1)*sb.Dx()/dw
			var r, g, b, n uint32
			for sy := y0; sy < max(y1, y0+1); sy++ {
				for sx := x0; sx < max(x1, x0+1); sx++ {
					pr, pg, pb, _ := img.At(sx, sy).RGBA()
					r, g, b, n = r+pr>>8, g+pg>>8, b+pb>>8, n+1
				}
			}
			out.SetRGBA(ox+x, oy+y, color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, out, &jpeg.Options{Quality: 92})
	return withJFIF(buf.Bytes())
}

// withJFIF adds the JFIF APP0 header Go's encoder leaves out, so the images are standard JFIF files like the
// ones the console's own items carry.
func withJFIF(b []byte) []byte {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 || (b[2] == 0xFF && b[3] == 0xE0) {
		return b
	}
	app0 := []byte{0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00}
	return append(append(append([]byte{}, b[:2]...), app0...), b[2:]...)
}

// defaultNewsImage is BCAT_NEWS_DIR/default.jpg (the logo), else a plain red square.
func defaultNewsImage() []byte {
	if b, err := os.ReadFile(filepath.Join(newsDir, "default.jpg")); err == nil && len(b) > 0 {
		return b
	}
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
	if topic == "nx_notice" {
		return "Nextendo Notices"
	}
	if c := channelByTopic(topic); c != nil {
		return c.Title
	}
	return "Nextendo Network"
}

// record is one item in the console's news record format (as the console's own local notices).
func (n newsFile) record() omap {
	r := omap{
		{"version", omap{{"format", 1}, {"semantics", 1}}},
		{"news_id", n.ID},
		{"published_at", n.when.Unix()},
		{"pickup_limit", n.when.Unix() + 14*24*3600}, // a time: featured (lock screen, top of News) until then
		{"priority", n.Priority},
		{"deletion_priority", 100},
		{"language", "en-US"},
		{"supported_languages", []string{"en-US"}},
		{"display_type", "NORMAL"},
		{"topic_id", n.Channel},
		{"no_photography", 0},
		{"surprise", 0},
		{"essential_pickup", omap{{"pickup_limit", n.when.Unix() + 14*24*3600}, {"priority_after", n.Priority}}}, // featured (lock screen)
		{"movie", 0},
		{"subject", omap{{"caption", 1}, {"text", n.Title}}},
		{"topic_name", topicName(n.Channel)},
	}
	// The channel's icon travels in its items (70x70, as the catalog's), and an opened item's "related
	// channels" section is its related_channels list (qlaunch 22.5.0 reads these five keys): its own channel,
	// then the news file's "related" ones.
	related := []any{}
	seen := map[string]bool{}
	for _, c := range append([]string{n.Channel}, n.Related...) {
		if seen[c] {
			continue
		}
		seen[c] = true
		related = append(related, omap{
			{"topic_id", c},
			{"topic_name", topicName(c)},
			{"topic_publisher", topicPublisher(c)},
			{"topic_image", channelIcon(c)},
			{"topic_important", 0},
		})
	}
	r = append(r, kv{"topic_image", channelIcon(n.Channel)}, kv{"list_image", n.img}, kv{"related_channels", related})
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

// summary is an item as a channel's page lists it (online_archives' summary_url), in the shape qlaunch
// (22.5.0) parses: no_photography, subject.text, movie, icon_image (70x70), list_image (358x201: any other
// size fails the whole summary) and url, the full item.
func (n newsFile) summary() omap {
	return omap{
		{"no_photography", 0},
		{"subject", omap{{"text", n.Title}}},
		{"movie", 0},
		{"icon_image", channelIcon(n.Channel)},
		{"list_image", fitJPEG(n.img, summaryImageW, summaryImageH)},
		{"url", newsDataURL(n)},
	}
}

const summaryImageW, summaryImageH = 358, 201

// newsPublisher is the publisher shown for a channel that names none.
const newsPublisher = "Nextendo Network"

func topicPublisher(topic string) string {
	if c := channelByTopic(topic); c != nil && c.Publisher != "" {
		return c.Publisher
	}
	return newsPublisher
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
	publisher, description := topicPublisher(topic), "News from the Nextendo Network."
	if c := channelByTopic(topic); c != nil && c.Description != "" {
		description = c.Description
	}
	return omap{
		{"topic_id", topic},
		{"name", topicName(topic)},
		{"publisher", publisher},
		{"description", description},
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
		catalog := []omap{}
		for _, c := range newsChannels() {
			catalog = append(catalog, channelEntry(c.Topic, items))
		}
		s.writeNewsContainer(w, mpack(catalog))
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
		// A channel's items, asked when a channel is opened (after its detail). The shape is what qlaunch
		// (22.5.0) reads: na_required and data_list, each item's languages carrying a summary_url. An array
		// here made it retry forever ("failed to load").
		topic := strings.TrimSuffix(strings.TrimPrefix(p, "/api/nx/v2/topics/"), "/online_archives")
		list := []any{}
		for _, n := range items {
			if n.Channel != topic {
				continue
			}
			list = append(list, omap{
				{"news_id", n.ID},
				{"version", omap{{"format", 1}, {"semantics", 1}}},
				{"default_language", "en-US"},
				{"languages", []any{omap{{"language", "en-US"}, {"summary_url", newsDataURL(n) + "/summary"}}}},
			})
		}
		s.writeNewsContainer(w, mpack(omap{{"na_required", false}, {"data_list", list}}))
		return true
	case strings.HasPrefix(p, "/api/nx/v1/topics/") && strings.HasSuffix(p, "/icon"):
		topic := strings.TrimSuffix(strings.TrimPrefix(p, "/api/nx/v1/topics/"), "/icon")
		s.writeNewsContainer(w, channelIcon(topic))
		return true
	case strings.HasPrefix(p, "/api/nx/v1/titles/") && strings.HasSuffix(p, "/topics"):
		// A title's channels, which the console subscribes to. The HOME menu's are the default feed: without
		// them a console whose news storage was cleared stops fetching the news lists.
		title := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(p, "/api/nx/v1/titles/"), "/topics"))
		topics := []string{}
		if title == "0100000000001000" {
			// nx_news and nx_notice are Nintendo's defaults; the custom channels marked default (the Nextendo
			// channel) are ours, added so every console follows them without the user finding them (not
			// something production does).
			topics = []string{"nx_news", "nx_notice"}
		}
		for _, c := range newsChannels() {
			if title == "0100000000001000" && c.Default {
				topics = append(topics, c.Topic)
			}
			for _, g := range c.Games {
				if g == title {
					topics = append(topics, c.Topic)
				}
			}
		}
		s.writeNewsContainer(w, mpack(topics))
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
	summary := len(parts) == 3 && parts[2] == "summary"
	if len(parts) != 2 && !summary {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.ParseUint(parts[1], 10, 32)
	for _, n := range loadNewsDir() {
		if n.Channel == parts[0] && uint64(n.ID) == id {
			if summary {
				s.writeNewsContainer(w, mpack(n.summary()))
			} else {
				s.writeNewsContainer(w, mpack(n.record()))
			}
			return
		}
	}
	http.NotFound(w, r)
}
