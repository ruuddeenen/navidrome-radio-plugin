// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package syncer implements the resumable plan/index/apply state machine that
// reconciles radio-browser stations with Navidrome's internet radio stations.
//
// It is intentionally free of PDK imports: the key-value store, HTTP fetch and
// Subsonic API are injected as interfaces so the logic is unit tested.
package syncer

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ruuddeenen/navidrome-radio-plugin/internal/filter"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/radiobrowser"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/settings"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/subsonic"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/template"
)

// Buckets is the number of shards the desired and existing sets are split into.
// Each shard is applied by its own task so a single task stays well below the
// 30s plugin-call timeout.
const Buckets = 256

// Entry is a desired station as persisted during the plan phase.
type Entry struct {
	Name string `json:"n"`
	URL  string `json:"u"`
	Home string `json:"h"`
}

// Existing is a station already present in Navidrome, keyed by stream URL.
type Existing struct {
	ID   string `json:"i"`
	Hash string `json:"x"`
}

// Store is the key-value state store (KVStore in production, a map in tests).
type Store interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Delete(key string)
	List(prefix string) ([]string, error)
	RemovePrefix(prefix string) error
}

// Fetcher pages through radio-browser stations.
type Fetcher interface {
	Page(offset int) ([]radiobrowser.Station, error)
}

// RadioAPI manages Navidrome internet radio stations.
type RadioAPI interface {
	List() ([]subsonic.Radio, error)
	Create(name, streamURL, homepageURL string) error
	Update(id, name, streamURL, homepageURL string) error
	Delete(id string) error
}

// Action is the next task to run (empty Kind means stop).
type Action struct {
	Kind   string
	Bucket int
}

// Runner executes one step of the sync state machine.
type Runner struct {
	S     settings.Settings
	Store Store
	Fetch Fetcher
	API   RadioAPI
	Log   func(format string, args ...any)
}

// Start resets the run state and returns the first action.
func (r *Runner) Start() (Action, error) {
	if err := r.Store.RemovePrefix("want:"); err != nil {
		return Action{}, err
	}
	if err := r.Store.RemovePrefix("have:"); err != nil {
		return Action{}, err
	}
	r.set("sync:offset", "0")
	r.set("sync:total", "0")
	r.set("sync:index_ok", "0")
	r.set("sync:created", "0")
	r.set("sync:updated", "0")
	r.set("sync:deleted", "0")
	r.set("sync:skipped", "0")
	r.set("sync:errors", "0")
	remaining := -1
	if r.S.BootstrapOpsPerRun > 0 {
		remaining = r.S.BootstrapOpsPerRun
	}
	r.setInt("sync:remaining", remaining)
	return Action{Kind: "plan"}, nil
}

// Plan fetches one page, filters it and stores the wanted entries in shards.
func (r *Runner) Plan() (Action, error) {
	offset := r.getInt("sync:offset", 0)
	page, err := r.Fetch.Page(offset)
	if err != nil {
		return Action{}, err
	}

	grouped := map[int][]Entry{}
	kept := 0
	total := r.getInt("sync:total", 0)
	maxed := false
	for _, st := range page {
		if r.S.MaxStations > 0 && total+kept >= r.S.MaxStations {
			maxed = true
			break
		}
		if decision := filter.Apply(r.S, st); !decision.Keep {
			continue
		}
		entry := r.entryFor(st)
		if entry.Name == "" || entry.URL == "" {
			continue
		}
		bucket := bucketOf(entry.URL)
		grouped[bucket] = append(grouped[bucket], entry)
		kept++
	}
	for bucket, entries := range grouped {
		if err := r.setJSON(fmt.Sprintf("want:%d:%d", bucket, offset), entries); err != nil {
			return Action{}, err
		}
	}
	r.setInt("sync:offset", offset+len(page))
	r.setInt("sync:total", total+kept)
	r.log("plan offset=%d fetched=%d kept=%d total=%d", offset, len(page), kept, total+kept)

	if maxed || len(page) < r.S.PageSize {
		return Action{Kind: "index"}, nil
	}
	return Action{Kind: "plan"}, nil
}

// Index reads the existing Navidrome stations and stores them in shards.
func (r *Runner) Index() (Action, error) {
	radios, err := r.API.List()
	if err != nil {
		r.log("index failed (%v); continuing create-only without pruning", err)
		r.set("sync:index_ok", "0")
		return Action{Kind: "apply", Bucket: 0}, nil
	}

	group := make([]map[string]Existing, Buckets)
	for i := range group {
		group[i] = map[string]Existing{}
	}
	for _, radio := range radios {
		if strings.TrimSpace(radio.StreamURL) == "" {
			continue
		}
		bucket := bucketOf(radio.StreamURL)
		group[bucket][radio.StreamURL] = Existing{ID: radio.ID, Hash: hashText(radio.Name + "|" + radio.HomepageURL)}
	}
	for bucket := 0; bucket < Buckets; bucket++ {
		if err := r.setJSON(fmt.Sprintf("have:%d", bucket), group[bucket]); err != nil {
			return Action{}, err
		}
	}
	r.set("sync:index_ok", "1")
	r.log("index: %d existing stations", len(radios))
	return Action{Kind: "apply", Bucket: 0}, nil
}

// Apply reconciles one bucket.
func (r *Runner) Apply(bucket int) (Action, error) {
	keys, err := r.Store.List(fmt.Sprintf("want:%d:", bucket))
	if err != nil {
		return Action{}, err
	}
	var want []Entry
	for _, key := range keys {
		if value, ok := r.Store.Get(key); ok {
			var entries []Entry
			if err := json.Unmarshal([]byte(value), &entries); err != nil {
				return Action{}, err
			}
			want = append(want, entries...)
		}
	}

	have := map[string]Existing{}
	if r.get("sync:index_ok") == "1" {
		if value, ok := r.Store.Get(fmt.Sprintf("have:%d", bucket)); ok {
			_ = json.Unmarshal([]byte(value), &have)
		}
	}

	created, updated, deleted, skipped, failed := 0, 0, 0, 0, 0
	for _, entry := range want {
		if r.budgetExhausted() {
			break
		}
		if existing, ok := have[entry.URL]; ok {
			delete(have, entry.URL)
			if existing.Hash != hashText(entry.Name+"|"+entry.Home) {
				if r.S.DryRun {
					updated++
				} else if err := r.API.Update(existing.ID, entry.Name, entry.URL, entry.Home); err != nil {
					failed++
					r.log("update %q failed: %v", entry.Name, err)
				} else {
					updated++
				}
				r.spend()
			} else {
				skipped++
			}
			continue
		}
		if r.S.DryRun {
			created++
		} else if err := r.API.Create(entry.Name, entry.URL, entry.Home); err != nil {
			if isDuplicate(err) {
				skipped++
			} else {
				failed++
				r.log("create %q failed: %v", entry.Name, err)
			}
		} else {
			created++
		}
		r.spend()
	}

	if r.S.PruneMissing && r.get("sync:index_ok") == "1" && !r.S.DryRun {
		for _, existing := range have {
			if r.budgetExhausted() {
				break
			}
			if err := r.API.Delete(existing.ID); err != nil {
				failed++
				r.log("delete %s failed: %v", existing.ID, err)
			} else {
				deleted++
			}
			r.spend()
		}
	}

	for _, key := range keys {
		r.Store.Delete(key)
	}
	r.Store.Delete(fmt.Sprintf("have:%d", bucket))

	r.add("sync:created", created)
	r.add("sync:updated", updated)
	r.add("sync:deleted", deleted)
	r.add("sync:skipped", skipped)
	r.add("sync:errors", failed)
	r.log("apply bucket %d: +%d ~%d -%d skip=%d err=%d", bucket, created, updated, deleted, skipped, failed)

	if r.budgetExhausted() {
		return Action{}, nil
	}
	if bucket+1 < Buckets {
		return Action{Kind: "apply", Bucket: bucket + 1}, nil
	}
	return Action{}, nil
}

// Summary returns the counters of the current (or last) run.
func (r *Runner) Summary() string {
	return fmt.Sprintf("created=%d updated=%d deleted=%d skipped=%d errors=%d planned=%d",
		r.getInt("sync:created", 0), r.getInt("sync:updated", 0), r.getInt("sync:deleted", 0),
		r.getInt("sync:skipped", 0), r.getInt("sync:errors", 0), r.getInt("sync:total", 0))
}

func (r *Runner) entryFor(st radiobrowser.Station) Entry {
	name := template.Render(r.S.NameTemplate, st.Field)
	if r.S.MaxNameLength > 0 {
		if runes := []rune(name); len(runes) > r.S.MaxNameLength {
			name = strings.TrimSpace(string(runes[:r.S.MaxNameLength]))
		}
	}
	streamURL := settings.EffectiveURL(st.Str("url"), st.Str("url_resolved"))
	home := strings.TrimSpace(st.Str("homepage"))
	if home == "" && r.S.HomepageFallback {
		home = originOf(streamURL)
	}
	return Entry{Name: name, URL: streamURL, Home: home}
}

func (r *Runner) budgetExhausted() bool {
	if r.S.BootstrapOpsPerRun <= 0 {
		return false
	}
	return r.getInt("sync:remaining", 0) <= 0
}

func (r *Runner) spend() {
	if r.S.BootstrapOpsPerRun <= 0 {
		return
	}
	if remaining := r.getInt("sync:remaining", 0); remaining > 0 {
		r.setInt("sync:remaining", remaining-1)
	}
}

func (r *Runner) get(key string) string {
	value, _ := r.Store.Get(key)
	return value
}

func (r *Runner) set(key, value string) { r.Store.Set(key, value) }

func (r *Runner) getInt(key string, fallback int) int {
	value, ok := r.Store.Get(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (r *Runner) setInt(key string, value int) { r.Store.Set(key, strconv.Itoa(value)) }

func (r *Runner) add(key string, delta int) {
	if delta != 0 {
		r.setInt(key, r.getInt(key, 0)+delta)
	}
}

func (r *Runner) setJSON(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	r.Store.Set(key, string(data))
	return nil
}

func (r *Runner) log(format string, args ...any) {
	if r.Log != nil {
		r.Log(format, args...)
	}
}

func fnv32(s string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	return hash
}

func bucketOf(streamURL string) int {
	return int(fnv32(streamURL) % uint32(Buckets))
}

func hashText(s string) string {
	return strconv.FormatUint(uint64(fnv32(s)), 16)
}

func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") ||
		strings.Contains(message, "already exists") ||
		strings.Contains(message, "constraint")
}

func originOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
