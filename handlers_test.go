package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mozilla-services/go-bouncer/bouncer"
	"github.com/stretchr/testify/assert"
)

var bouncerHandler *BouncerHandler

func init() {
	testDB, err := bouncer.NewDB("root@tcp(127.0.0.1:3306)/bouncer_test")
	if err != nil {
		log.Fatal(err)
	}

	bouncerHandler = &BouncerHandler{db: testDB}
}

func TestBouncerHandlerParams(t *testing.T) {
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://test/?os=mac&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "http://www.mozilla.org/", w.HeaderMap.Get("Location"))
}

func TestBouncerHandlerPrintQuery(t *testing.T) {
	w := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "http://test/?product=firefox-latest&os=osx&lang=en-US&print=yes", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "http://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/mac/en-US/Firefox%2039.0.dmg", w.Body.String())
}

func TestBouncerHandlerValid(t *testing.T) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://test/?product=firefox-latest&os=osx&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "http://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/mac/en-US/Firefox%2039.0.dmg", w.HeaderMap.Get("Location"))

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://test/?product=firefox-latest&os=win64&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "http://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/win32/en-US/Firefox%20Setup%2039.0.exe", w.HeaderMap.Get("Location"))

	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://test/?product=Firefox-SSL&os=win64&lang=en-US", nil)
	assert.NoError(t, err)

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "https://download-installer.cdn.mozilla.net/pub/firefox/releases/39.0/win32/en-US/Firefox%20Setup%2039.0.exe", w.HeaderMap.Get("Location"))

	// Test Windows XP
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://test/?product=Firefox-SSL&os=win&lang=en-US", nil)
	assert.NoError(t, err)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows; U; MSIE 6.0; Windows NT 5.1; SV1; .NET CLR 2.0.50727)")

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "https://download-installer.cdn.mozilla.net/pub/firefox/releases/43.0.1/win32/en-US/Firefox%20Setup%2043.0.1.exe", w.HeaderMap.Get("Location"))

	// Test Windows XP 64 bit
	w = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://test/?product=Firefox-SSL&os=win64&lang=en-US", nil)
	assert.NoError(t, err)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows; U; MSIE 6.0; Windows NT 5.1; SV1; .NET CLR 2.0.50727)")

	bouncerHandler.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "https://download-installer.cdn.mozilla.net/pub/firefox/releases/43.0.1/win64/en-US/Firefox%20Setup%2043.0.1.exe", w.HeaderMap.Get("Location"))
}

func TestIsWindowsXPUserAgent(t *testing.T) {
	uas := []struct {
		UA   string
		IsXP bool
	}{
		{"Mozilla/5.0 (Windows NT 5.1; rv:31.0) Gecko/20100101 Firefox/31.0", true},                                            // firefox XP
		{"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:31.0) Gecko/20130401 Firefox/31.0", false},                                    // firefox non-XP
		{"Mozilla/5.0 (Windows NT 5.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2224.3 Safari/537.36", true},         // Chrome XP
		{"Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2225.0 Safari/537.36", false}, // Chrome non-XP
		{"Mozilla/5.0 (Windows; U; MSIE 6.0; Windows NT 5.1; SV1; .NET CLR 2.0.50727)", true},                                  // IE XP
		{"Mozilla/4.0 (compatible; MSIE 6.1; Windows XP)", true},                                                               // IE XP
		{"Mozilla/5.0 (Windows; U; MSIE 7.0; Windows NT 6.0; en-US)", false},                                                   // IE non-XP
	}
	for _, ua := range uas {
		assert.Equal(t, ua.IsXP, isWindowsXPUserAgent(ua.UA))
	}
}

func TestSha1Product(t *testing.T) {
	assert.Equal(t, "firefox-43.0.1", sha1Product("firefox-latest"))
	assert.Equal(t, "firefox", sha1Product("firefox"))
	assert.Equal(t, "firefox-42.0.0-ssl", sha1Product("firefox-42.0.0-ssl"))
	assert.Equal(t, "firefox-43.0.1-ssl", sha1Product("firefox-43.0.2-ssl"))
	assert.Equal(t, "firefox-43.0.1-ssl", sha1Product("firefox-44.0.0-ssl"))

	assert.Equal(t, "firefox-42.0.0", sha1Product("firefox-42.0.0"))
	assert.Equal(t, "firefox-43.0.1", sha1Product("firefox-43.0.1"))
	assert.Equal(t, "firefox-43.0.1", sha1Product("firefox-43.0.2"))

	assert.Equal(t, "firefox-42.0.0-stub", sha1Product("firefox-42.0.0-stub"))
	assert.Equal(t, "firefox-43.0.1-stub", sha1Product("firefox-43.0.1-stub"))
	assert.Equal(t, "firefox-43.0.1-stub", sha1Product("firefox-43.0.2-stub"))

	assert.Equal(t, "firefox-42.0.0-complete", sha1Product("firefox-42.0.0-complete"))
	assert.Equal(t, "firefox-43.0.1-partial-41.0.2build1", sha1Product("firefox-43.0.1-partial-41.0.2build1"))
	assert.Equal(t, "firefox-43.0.2-complete", sha1Product("firefox-43.0.2-complete"))
	assert.Equal(t, "firefox-44.0-complete", sha1Product("firefox-44.0-complete"))

	assert.Equal(t, "firefox-45.0b1-complete", sha1Product("firefox-45.0b1-complete"))
	assert.Equal(t, "firefox-44.0b1-stub", sha1Product("firefox-45.0b1-stub"))
	assert.Equal(t, "firefox-44.0b1-ssl", sha1Product("firefox-45.0b1-ssl"))
	assert.Equal(t, "firefox-44.0b1-stub", sha1Product("firefox-beta-stub"))
	assert.Equal(t, "firefox-44.0b1", sha1Product("firefox-beta"))
	assert.Equal(t, "firefox-43.0b1", sha1Product("firefox-43.0b1"))
	assert.Equal(t, "firefox-44.0b1", sha1Product("firefox-45.0b2"))
	assert.Equal(t, "firefox-44.0b1", sha1Product("firefox-44.0b2"))

	assert.Equal(t, "firefox-35.0.1esr", sha1Product("firefox-35.0.1esr"))
	assert.Equal(t, "firefox-38.5.0esr", sha1Product("firefox-38.5.0esr"))
	assert.Equal(t, "firefox-38.5.1esr", sha1Product("firefox-38.5.2esr"))
	assert.Equal(t, "firefox-38.5.1esr", sha1Product("firefox-38.5.3esr"))
	assert.Equal(t, "firefox-38.5.1esr", sha1Product("firefox-38.6.3esr"))
	assert.Equal(t, "firefox-38.5.1esr", sha1Product("firefox-40.0.0esr"))
}

func BenchmarkSha1Product(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sha1Product("firefox-43.0.0")
		sha1Product("firefox-44.0b1")
	}
}
