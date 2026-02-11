package cookies

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"
)

// Jar wraps http.CookieJar with thread-safe access and manual management.
type Jar struct {
	jar  http.CookieJar
	mu   sync.RWMutex
	urls map[string]*url.URL // track domains we've seen
}

// New creates a new cookie jar.
func New() *Jar {
	j, _ := cookiejar.New(nil)
	return &Jar{
		jar:  j,
		urls: make(map[string]*url.URL),
	}
}

// GetJar returns the underlying http.CookieJar for use with http.Client.
func (j *Jar) GetJar() http.CookieJar {
	return j.jar
}

// Cookies returns cookies for a URL.
func (j *Jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.jar.Cookies(u)
}

// SetCookies adds cookies for a URL.
func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.urls[u.Host] = u
	j.jar.SetCookies(u, cookies)
}

// AllCookies returns all known cookies keyed by host.
func (j *Jar) AllCookies() map[string][]*http.Cookie {
	j.mu.RLock()
	defer j.mu.RUnlock()
	result := make(map[string][]*http.Cookie)
	for host, u := range j.urls {
		cookies := j.jar.Cookies(u)
		if len(cookies) > 0 {
			result[host] = cookies
		}
	}
	return result
}

// Clear removes all cookies by replacing the underlying jar.
func (j *Jar) Clear() {
	j.mu.Lock()
	defer j.mu.Unlock()
	newJar, _ := cookiejar.New(nil)
	j.jar = newJar
	j.urls = make(map[string]*url.URL)
}

// persistedCookie is a JSON-serializable cookie format.
type persistedCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HTTPOnly bool      `json:"http_only,omitempty"`
	SameSite string    `json:"same_site,omitempty"`
}

type persistedJar struct {
	Cookies map[string][]persistedCookie `json:"cookies"`
}

// SaveToFile persists all cookies to a JSON file.
func (j *Jar) SaveToFile(path string) error {
	j.mu.RLock()
	defer j.mu.RUnlock()

	data := persistedJar{Cookies: make(map[string][]persistedCookie)}
	for host, u := range j.urls {
		cookies := j.jar.Cookies(u)
		for _, c := range cookies {
			data.Cookies[host] = append(data.Cookies[host], persistedCookie{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   c.Domain,
				Path:     c.Path,
				Expires:  c.Expires,
				Secure:   c.Secure,
				HTTPOnly: c.HttpOnly,
			})
		}
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// LoadFromFile loads cookies from a JSON file.
func (j *Jar) LoadFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no cookie file yet
		}
		return err
	}

	var data persistedJar
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	for host, cookies := range data.Cookies {
		u := &url.URL{Scheme: "https", Host: host}
		j.urls[host] = u
		var httpCookies []*http.Cookie
		for _, c := range cookies {
			httpCookies = append(httpCookies, &http.Cookie{
				Name:     c.Name,
				Value:    c.Value,
				Domain:   c.Domain,
				Path:     c.Path,
				Expires:  c.Expires,
				Secure:   c.Secure,
				HttpOnly: c.HTTPOnly,
			})
		}
		if len(httpCookies) > 0 {
			j.jar.SetCookies(u, httpCookies)
		}
	}
	return nil
}

// RemoveCookie removes a specific cookie by domain and name.
// It does this by re-setting all cookies except the matching one.
func (j *Jar) RemoveCookie(domain, name string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	u, ok := j.urls[domain]
	if !ok {
		return
	}

	cookies := j.jar.Cookies(u)
	var keep []*http.Cookie
	for _, c := range cookies {
		if c.Name != name {
			keep = append(keep, c)
		}
	}

	// Replace the jar and re-add all cookies except the removed one
	newJar, _ := cookiejar.New(nil)
	for host, storedURL := range j.urls {
		if host == domain {
			if len(keep) > 0 {
				newJar.SetCookies(storedURL, keep)
			}
		} else {
			existing := j.jar.Cookies(storedURL)
			if len(existing) > 0 {
				newJar.SetCookies(storedURL, existing)
			}
		}
	}
	j.jar = newJar

	// Clean up the domain entry if no cookies remain
	if len(keep) == 0 {
		delete(j.urls, domain)
	}
}
