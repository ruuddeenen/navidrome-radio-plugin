// SPDX-License-Identifier: GPL-3.0-or-later
//
// Package subsonic wraps the Navidrome Subsonic API endpoints used to manage
// internet radio stations. The actual transport is injected as a Caller so the
// package stays testable and PDK-free.
package subsonic

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Radio is an internet radio station as returned by getInternetRadioStations.
type Radio struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	StreamURL   string `json:"streamUrl"`
	HomepageURL string `json:"homePageUrl"`
}

// Caller executes a Subsonic API request (path + query) and returns JSON.
type Caller func(uri string) (string, error)

// Client talks to the Subsonic radio endpoints as a given user.
type Client struct {
	Call Caller
	User string
}

type subsonicError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	SubsonicResponse struct {
		Status                string         `json:"status"`
		Error                 *subsonicError `json:"error"`
		InternetRadioStations struct {
			Radio []Radio `json:"internetRadioStation"`
		} `json:"internetRadioStations"`
	} `json:"subsonic-response"`
}

// List returns all internet radio stations known to Navidrome.
func (c *Client) List() ([]Radio, error) {
	uri, err := c.endpoint("getInternetRadioStations", url.Values{})
	if err != nil {
		return nil, err
	}
	body, err := c.Call(uri)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return nil, fmt.Errorf("parsing getInternetRadioStations: %w", err)
	}
	if e := env.SubsonicResponse.Error; e != nil {
		return nil, fmt.Errorf("subsonic error %d: %s", e.Code, e.Message)
	}
	return env.SubsonicResponse.InternetRadioStations.Radio, nil
}

// Create adds a new internet radio station.
func (c *Client) Create(name, streamURL, homepageURL string) error {
	return c.write("createInternetRadioStation", name, streamURL, homepageURL, "")
}

// Update changes an existing internet radio station.
func (c *Client) Update(id, name, streamURL, homepageURL string) error {
	return c.write("updateInternetRadioStation", name, streamURL, homepageURL, id)
}

// Delete removes an internet radio station by ID.
func (c *Client) Delete(id string) error {
	params := url.Values{}
	params.Set("id", id)
	uri, err := c.endpoint("deleteInternetRadioStation", params)
	if err != nil {
		return err
	}
	body, err := c.Call(uri)
	if err != nil {
		return err
	}
	return responseError(body)
}

func (c *Client) write(op, name, streamURL, homepageURL, id string) error {
	params := url.Values{}
	params.Set("name", name)
	params.Set("streamUrl", streamURL)
	if homepageURL != "" {
		params.Set("homepageUrl", homepageURL)
	}
	if id != "" {
		params.Set("id", id)
	}
	uri, err := c.endpoint(op, params)
	if err != nil {
		return err
	}
	body, err := c.Call(uri)
	if err != nil {
		return err
	}
	return responseError(body)
}

func (c *Client) endpoint(name string, params url.Values) (string, error) {
	if c.Call == nil {
		return "", fmt.Errorf("no subsonic caller configured")
	}
	if c.User == "" {
		return "", fmt.Errorf("no admin user configured for subsonic calls")
	}
	params.Set("u", c.User)
	return name + "?" + params.Encode(), nil
}

func responseError(body string) error {
	var env envelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return nil // not a JSON error envelope; treat as success
	}
	if e := env.SubsonicResponse.Error; e != nil {
		return fmt.Errorf("subsonic error %d: %s", e.Code, e.Message)
	}
	return nil
}
