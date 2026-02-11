package cookies

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
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
