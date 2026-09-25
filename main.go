// SPDX-License-Identifier: GPL-3.0-or-later
//
// Entry point for the Automatic Radio Sync plugin.
//
// A scheduled task periodically fetches internet radio stations from
// radio-browser.info, applies the configured filters and creates/updates/removes
// the matching internet radio stations in Navidrome via the Subsonic API.
package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/taskworker"

	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/radiobrowser"
	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/settings"
	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/subsonic"
	"github.com/ruuddeenen/navidrome-radio-sync-plugin/internal/syncer"
)

const (
	queueName   = "radio-sync"
	scheduleID  = "automatic-radio-sync"
	payloadSync = "sync"

	kindStart = "start"
	kindPlan  = "plan"
	kindIndex = "index"
	kindApply = "apply"
)

type plugin struct{}

func init() {
	lifecycle.Register(&plugin{})
	scheduler.Register(&plugin{})
	taskworker.Register(&plugin{})
}

// ---------- lifecycle ----------

func (p *plugin) OnInit() error {
	s := loadSettings()

	config := host.QueueConfig{
		Concurrency: 1,
		MaxRetries:  2,
		BackoffMs:   5000,
		RetentionMs: 3600000,
	}
	if s.TaskDelayMs > 0 {
		config.DelayMs = int64(s.TaskDelayMs)
	}
	_ = host.TaskCreateQueue(queueName, config)

	if _, err := host.SchedulerScheduleRecurring(s.SyncCron, payloadSync, scheduleID); err != nil {
		pdk.Log(pdk.LogError, "failed to schedule radio sync: "+err.Error())
		return err
	}

	// "Sync after saving settings": when enabled, run a one-time sync whenever
	// the config changes. Saving settings unloads and reloads the plugin (which
	// calls OnInit), so compare a fingerprint of the current config with the last
	// seen one. This keeps the toggle stateless in the plugin (config is
	// read-only) and does not run on restarts without a config change.
	store := kvStore{}
	fingerprint := configFingerprint()
	if s.RunOnSave {
		if last, _ := store.Get("trigger:config_fingerprint"); last != fingerprint {
			if _, err := host.SchedulerScheduleOneTime(1, payloadSync, scheduleID+"-manual"); err != nil {
				pdk.Log(pdk.LogWarn, "failed to schedule manual sync: "+err.Error())
			} else {
				pdk.Log(pdk.LogInfo, "one-time sync triggered by settings change")
			}
		}
	}
	store.Set("trigger:config_fingerprint", fingerprint)

	pdk.Log(pdk.LogInfo, "automatic-radio-sync ready; cron="+s.SyncCron)
	return nil
}

// ---------- scheduler ----------

func (p *plugin) OnCallback(req scheduler.SchedulerCallbackRequest) error {
	if req.Payload != payloadSync {
		return nil
	}
	return enqueue(kindStart, 0)
}

// ---------- task worker ----------

type taskPayload struct {
	Kind   string `json:"kind"`
	Bucket int    `json:"bucket,omitempty"`
}

func (p *plugin) OnTaskExecute(req taskworker.TaskExecuteRequest) (string, error) {
	var payload taskPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return "", err
	}

	runner, err := buildRunner(loadSettings())
	if err != nil {
		return "", err
	}

	var action syncer.Action
	switch payload.Kind {
	case kindStart:
		action, err = runner.Start()
	case kindPlan:
		action, err = runner.Plan()
	case kindIndex:
		action, err = runner.Index()
	case kindApply:
		action, err = runner.Apply(payload.Bucket)
	default:
		return "unknown task kind: " + payload.Kind, nil
	}
	if err != nil {
		return "", err
	}
	return followUp(runner, action)
}

func followUp(runner *syncer.Runner, action syncer.Action) (string, error) {
	if action.Kind == "" {
		summary := runner.Summary()
		pdk.Log(pdk.LogInfo, "radio sync finished: "+summary)
		return summary, nil
	}
	if err := enqueue(action.Kind, action.Bucket); err != nil {
		return "", err
	}
	return action.Kind, nil
}

func enqueue(kind string, bucket int) error {
	data, err := json.Marshal(taskPayload{Kind: kind, Bucket: bucket})
	if err != nil {
		return err
	}
	_, err = host.TaskEnqueue(queueName, data)
	return err
}

func buildRunner(s settings.Settings) (*syncer.Runner, error) {
	admin, err := resolveAdmin(s)
	if err != nil {
		return nil, err
	}
	return &syncer.Runner{
		S:     s,
		Store: kvStore{},
		Fetch: &radiobrowser.Client{
			Doer:       hostDoer{},
			BaseURL:    s.BaseURL,
			PageSize:   s.PageSize,
			HideBroken: s.HideBroken,
			Order:      s.Order,
			Reverse:    s.Reverse,
		},
		API: &subsonic.Client{
			User: admin,
			Call: func(uri string) (string, error) { return host.SubsonicAPICall(uri) },
		},
		Log: func(format string, args ...any) {
			pdk.Log(pdk.LogInfo, fmt.Sprintf(format, args...))
		},
	}, nil
}

func resolveAdmin(s settings.Settings) (string, error) {
	if s.AdminUser != "" {
		return s.AdminUser, nil
	}
	admins, err := host.UsersGetAdmins()
	if err != nil {
		return "", err
	}
	if len(admins) == 0 {
		return "", fmt.Errorf("no admin user available for subsonic calls")
	}
	return admins[0].UserName, nil
}

// ---------- adapters ----------

type hostDoer struct{}

func (hostDoer) Do(method, url string, headers map[string]string, timeoutMs int32) (int, []byte, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:    method,
		URL:       url,
		Headers:   headers,
		TimeoutMs: timeoutMs,
	})
	if err != nil {
		return 0, nil, err
	}
	return int(resp.StatusCode), resp.Body, nil
}

type kvStore struct{}

func (kvStore) Get(key string) (string, bool) {
	value, exists, err := host.KVStoreGet(key)
	if err != nil || !exists {
		return "", false
	}
	return string(value), true
}

func (kvStore) Set(key, value string) { _ = host.KVStoreSet(key, []byte(value)) }

func (kvStore) Delete(key string) { _ = host.KVStoreDelete(key) }

func (kvStore) List(prefix string) ([]string, error) { return host.KVStoreList(prefix) }

func (kvStore) RemovePrefix(prefix string) error {
	_, err := host.KVStoreDeleteByPrefix(prefix)
	return err
}

func loadSettings() settings.Settings {
	return settings.Load(func(key string) (string, bool) {
		return host.ConfigGet(key)
	})
}

// configFingerprint returns a stable hash of the current plugin configuration.
func configFingerprint() string {
	keys := host.ConfigKeys("")
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		value, _ := host.ConfigGet(key)
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(value)
		builder.WriteByte('\n')
	}
	data := builder.String()
	var hash uint32 = 2166136261
	for i := 0; i < len(data); i++ {
		hash ^= uint32(data[i])
		hash *= 16777619
	}
	return strconv.FormatUint(uint64(hash), 16)
}

func main() {}
