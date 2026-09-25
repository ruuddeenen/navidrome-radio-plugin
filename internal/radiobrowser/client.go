// SPDX-License-Identifier: GPL-3.0-or-later
package radiobrowser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DefaultMirrors is the list of public radio-browser.info mirrors, tried in order.
var DefaultMirrors = []string{
	"https://de1.api.radio-browser.info",
	"https://de2.api.radio-browser.info",
	"https://nl1.api.radio-browser.info",
	"https://at1.api.radio-browser.info",
	"https://fr1.api.radio-browser.info",
}

const defaultUserAgent = "navidrome-radio-plugin/0.1 (+https://github.com/ruuddeenen/navidrome-radio-plugin)"

// Doer performs an HTTP request and returns status code and body.
type Doer interface {
	Do(method, url string, headers map[string]string, timeoutMs int32) (status int, body []byte, err error)
}

// Client fetches stations from a radio-browser mirror.
type Client struct {
	Doer       Doer
	BaseURL    string
	Mirrors    []string
	UserAgent  string
	PageSize   int
	HideBroken bool
	Order      string
	Reverse    bool
	TimeoutMs  int32
}

// Page fetches one page of stations starting at offset. It rotates through the
// configured mirrors (with retries) and returns the parsed stations.
func (c *Client) Page(offset int) ([]Station, error) {
	if c.Doer == nil {
		return nil, fmt.Errorf("no HTTP doer configured")
	}
	pageSize := c.PageSize
	if pageSize <= 0 {
		pageSize = 2000
	}
	timeout := c.TimeoutMs
	if timeout <= 0 {
		timeout = 60000
	}
	ua := c.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	params := url.Values{}
	params.Set("limit", strconv.Itoa(pageSize))
	params.Set("offset", strconv.Itoa(offset))
	if c.HideBroken {
		params.Set("hidebroken", "true")
	}
	if c.Order != "" {
		params.Set("order", c.Order)
	}
	if c.Reverse {
		params.Set("reverse", "true")
	}
	query := params.Encode()

	bases := c.baseURLs()
	var lastErr error
	for attempt := 0; attempt < len(bases)*2; attempt++ {
		base := bases[attempt%len(bases)]
		requestURL := strings.TrimRight(base, "/") + "/json/stations/search?" + query
		headers := map[string]string{"User-Agent": ua, "Accept": "application/json"}
		status, body, err := c.Doer.Do("GET", requestURL, headers, timeout)
		if err == nil && status == 200 {
			return ParseStations(body)
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("HTTP %d from %s", status, base)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no radio-browser mirrors configured")
	}
	return nil, lastErr
}

func (c *Client) baseURLs() []string {
	if strings.TrimSpace(c.BaseURL) != "" {
		return []string{strings.TrimSpace(c.BaseURL)}
	}
	if len(c.Mirrors) > 0 {
		return c.Mirrors
	}
	return DefaultMirrors
}
