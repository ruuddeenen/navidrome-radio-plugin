// SPDX-License-Identifier: GPL-3.0-or-later
//
// Entry point for the Navidrome Radio Plugin.
//
// A scheduled task periodically fetches internet radio stations from
// radio-browser.info, applies the configured filters and creates/updates/removes
// the matching internet radio stations in Navidrome via the Subsonic API.
//
// The current implementation contains the dry-run probe (step 1): it fetches a
// few pages, applies the filters and logs the resulting names without writing to
// Navidrome.
package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/taskworker"

	"github.com/ruuddeenen/navidrome-radio-plugin/internal/filter"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/radiobrowser"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/settings"
	"github.com/ruuddeenen/navidrome-radio-plugin/internal/template"
)

const (
	queueName   = "radio-sync"
	scheduleID  = "navidrome-radio-sync"
	payloadSync = "sync"

	kindProbe = "probe"
)

// probePages bounds the dry-run probe so a first test stays fast.
const probePages = 3

type plugin struct{}

func init() {
	lifecycle.Register(&plugin{})
	scheduler.Register(&plugin{})
	taskworker.Register(&plugin{})
}

// ---------- lifecycle ----------

func (p *plugin) OnInit() error {
	s := loadSettings()

	_ = host.TaskCreateQueue(queueName, host.QueueConfig{
		Concurrency: 1,
		MaxRetries:  2,
		BackoffMs:   5000,
		RetentionMs: 3600000,
	})

	if _, err := host.SchedulerScheduleRecurring(s.SyncCron, payloadSync, scheduleID); err != nil {
		pdk.Log(pdk.LogError, "failed to schedule radio sync: "+err.Error())
		return err
	}
	if _, err := host.SchedulerScheduleOneTime(20, payloadSync, scheduleID+"-initial"); err != nil {
		pdk.Log(pdk.LogWarn, "failed to schedule initial radio sync: "+err.Error())
	}
	pdk.Log(pdk.LogInfo, "navidrome-radio-plugin ready; cron="+s.SyncCron)
	return nil
}

// ---------- scheduler ----------

func (p *plugin) OnCallback(req scheduler.SchedulerCallbackRequest) error {
	if req.Payload != payloadSync {
		return nil
	}
	payload, err := json.Marshal(taskPayload{Kind: kindProbe})
	if err != nil {
		return err
	}
	_, err = host.TaskEnqueue(queueName, payload)
	return err
}

// ---------- task worker ----------

type taskPayload struct {
	Kind string `json:"kind"`
	Page int    `json:"page,omitempty"`
}

func (p *plugin) OnTaskExecute(req taskworker.TaskExecuteRequest) (string, error) {
	var payload taskPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return "", err
	}
	switch payload.Kind {
	case kindProbe:
		return runProbe()
	default:
		return "unknown task kind: " + payload.Kind, nil
	}
}

// runProbe fetches a few pages, applies the filters and logs the resulting
// names. It does not touch Navidrome.
func runProbe() (string, error) {
	s := loadSettings()
	client := &radiobrowser.Client{
		Doer:       hostDoer{},
		BaseURL:    s.BaseURL,
		PageSize:   s.PageSize,
		HideBroken: s.HideBroken,
		Order:      s.Order,
		Reverse:    s.Reverse,
	}

	total := 0
	kept := 0
	reasons := map[string]int{}
	var samples []string

	for page := 0; page < probePages; page++ {
		stations, err := client.Page(page * s.PageSize)
		if err != nil {
			return "", err
		}
		total += len(stations)
		for _, st := range stations {
			decision := filter.Apply(s, st)
			if !decision.Keep {
				reasons[decision.Reason]++
				continue
			}
			kept++
			if len(samples) < 10 {
				samples = append(samples, template.Render(s.NameTemplate, st.Field))
			}
		}
		if len(stations) < s.PageSize {
			break
		}
	}

	summary := fmt.Sprintf("dry-run probe: fetched=%d kept=%d", total, kept)
	pdk.Log(pdk.LogInfo, summary)
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pdk.Log(pdk.LogInfo, fmt.Sprintf("  filtered %s=%d", k, reasons[k]))
	}
	for _, name := range samples {
		pdk.Log(pdk.LogInfo, "  sample: "+name)
	}
	return summary, nil
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

func loadSettings() settings.Settings {
	return settings.Load(func(key string) (string, bool) {
		return host.ConfigGet(key)
	})
}

func main() {}
