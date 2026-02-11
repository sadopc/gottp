package cookies

import (
	"net/http"
	"net/url"
	"testing"
)

func TestJar_SetAndGetCookies(t *testing.T) {
	jar := New()
	u, _ := url.Parse("https://example.com")

	jar.SetCookies(u, []*http.Cookie{
		{Name: "session", Value: "abc123"},
		{Name: "token", Value: "xyz789"},
	})

	cookies := jar.Cookies(u)
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	found := make(map[string]string)
	for _, c := range cookies {
		found[c.Name] = c.Value
	}
	if found["session"] != "abc123" {
		t.Errorf("expected session=abc123, got %s", found["session"])
	}
	if found["token"] != "xyz789" {
		t.Errorf("expected token=xyz789, got %s", found["token"])
	}
}

func TestJar_AllCookies(t *testing.T) {
	jar := New()

	u1, _ := url.Parse("https://example.com")
	u2, _ := url.Parse("https://other.com")

	jar.SetCookies(u1, []*http.Cookie{{Name: "a", Value: "1"}})
	jar.SetCookies(u2, []*http.Cookie{{Name: "b", Value: "2"}})

	all := jar.AllCookies()
	if len(all) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(all))
	}
	if len(all["example.com"]) != 1 {
		t.Errorf("expected 1 cookie for example.com, got %d", len(all["example.com"]))
	}
	if len(all["other.com"]) != 1 {
		t.Errorf("expected 1 cookie for other.com, got %d", len(all["other.com"]))
	}
}

func TestJar_Clear(t *testing.T) {
	jar := New()
	u, _ := url.Parse("https://example.com")

	jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "abc"}})
	jar.Clear()

	cookies := jar.Cookies(u)
	if len(cookies) != 0 {
		t.Errorf("expected 0 cookies after clear, got %d", len(cookies))
	}

	all := jar.AllCookies()
	if len(all) != 0 {
		t.Errorf("expected 0 domains after clear, got %d", len(all))
	}
}

func TestJar_RemoveCookie(t *testing.T) {
	jar := New()
	u, _ := url.Parse("https://example.com")

	jar.SetCookies(u, []*http.Cookie{
		{Name: "keep", Value: "yes"},
		{Name: "remove", Value: "no"},
	})

	jar.RemoveCookie("example.com", "remove")

	cookies := jar.Cookies(u)
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie after remove, got %d", len(cookies))
	}
	if cookies[0].Name != "keep" {
		t.Errorf("expected remaining cookie to be 'keep', got %s", cookies[0].Name)
	}
}

func TestJar_RemoveCookie_NonExistentDomain(t *testing.T) {
	jar := New()
	// Should not panic
	jar.RemoveCookie("nonexistent.com", "cookie")
}

func TestJar_GetJar(t *testing.T) {
	jar := New()
	if jar.GetJar() == nil {
		t.Error("GetJar() should not return nil")
	}
}
