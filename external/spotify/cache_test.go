package spotify

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestPlaylistCacheServesDiskWithoutAPI(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		var body string
		switch req.URL.Path {
		case "/v1/me":
			body = `{"id":"me"}`
		case "/v1/me/tracks":
			body = `{"total":1}`
		case "/v1/me/playlists":
			body = `{"items":[{"id":"pl1","name":"Cached","snapshot_id":"snap1","owner":{"id":"me"},"items":{"total":1}}],"total":1}`
		default:
			return nil, fmt.Errorf("unexpected Spotify API path %q", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	sess := &Session{tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"})}
	first := New(sess, "client", 320)
	first.cacheDir = dir
	if _, err := first.Playlists(); err != nil {
		t.Fatal(err)
	}
	if calls == 0 {
		t.Fatal("first Playlists() made no API calls")
	}

	afterFirst := calls
	second := New(sess, "client", 320)
	second.cacheDir = dir
	got, err := second.Playlists()
	if err != nil {
		t.Fatal(err)
	}
	if calls != afterFirst {
		t.Fatalf("second Playlists() made API calls, want disk cache only (calls %d -> %d)", afterFirst, calls)
	}
	if len(got) < 2 || got[1].ID != "pl1" {
		t.Fatalf("cached playlists = %#v, want pl1 from disk", got)
	}

	second.Refresh()
	if _, err := second.Playlists(); err != nil {
		t.Fatal(err)
	}
	if calls <= afterFirst {
		t.Fatalf("Refresh()+Playlists() calls = %d, want a new API fetch", calls)
	}
}

func TestTrackCacheServesDiskWithoutAPI(t *testing.T) {
	dir := t.TempDir()
	trackCalls := 0
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.Path {
		case "/v1/playlists/pl1/items":
			trackCalls++
			body = `{"items":[{"track":{"id":"t1","name":"Song","type":"track","uri":"spotify:track:t1","duration_ms":180000,"artists":[{"name":"A"}]}}],"total":1}`
		default:
			return nil, fmt.Errorf("unexpected Spotify API path %q", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	sess := &Session{tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"})}
	first := New(sess, "client", 320)
	first.cacheDir = dir
	got, err := first.Tracks("pl1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Song" {
		t.Fatalf("tracks = %#v", got)
	}
	if trackCalls != 1 {
		t.Fatalf("track API calls = %d, want 1", trackCalls)
	}

	second := New(sess, "client", 320)
	second.cacheDir = dir
	got, err = second.Tracks("pl1")
	if err != nil {
		t.Fatal(err)
	}
	if trackCalls != 1 {
		t.Fatalf("second Tracks() hit API (%d calls), want disk cache", trackCalls)
	}
	if len(got) != 1 || got[0].Title != "Song" {
		t.Fatalf("cached tracks = %#v", got)
	}
}
