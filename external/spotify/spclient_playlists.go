package spotify

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/bjarneo/cliamp/applog"
	"github.com/bjarneo/cliamp/playlist"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	playlist4pb "github.com/devgianlu/go-librespot/proto/spotify/playlist4"
	"google.golang.org/protobuf/proto"
)

func (s *Session) fetchSelectedList(ctx context.Context, path string) (*playlist4pb.SelectedListContent, error) {
	if s == nil {
		return nil, fmt.Errorf("spotify: session closed")
	}
	s.mu.RLock()
	sess := s.sess
	s.mu.RUnlock()
	if sess == nil {
		return nil, fmt.Errorf("spotify: session closed")
	}
	resp, err := sess.Spclient().Request(ctx, "GET", path, url.Values{
		"decorate": {"revision,attributes,length,owner,capabilities"},
	}, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("spotify: %s: status %d: %s", path, resp.StatusCode, body)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, err
	}
	var list playlist4pb.SelectedListContent
	if err := proto.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("spotify: decode %s: %w", path, err)
	}
	return &list, nil
}

func (p *SpotifyProvider) fetchPlaylistsSpclient(ctx context.Context) ([]playlist.PlaylistInfo, error) {
	user := p.currentUserID(ctx)
	if user == "" {
		return nil, fmt.Errorf("spotify: no username for librespot playlist list")
	}
	list, err := p.session.fetchSelectedList(ctx, "/playlist/v2/user/"+url.PathEscape(user)+"/rootlist")
	if err != nil {
		return nil, err
	}
	return playlistsFromRootlist(user, list), nil
}

func playlistsFromRootlist(userID string, list *playlist4pb.SelectedListContent) []playlist.PlaylistInfo {
	if list == nil || list.GetContents() == nil {
		return nil
	}
	items := list.GetContents().GetItems()
	metas := list.GetContents().GetMetaItems()
	out := []playlist.PlaylistInfo{{
		ID:      "YOUR MUSIC",
		Name:    "Your Music",
		Section: "Library",
	}}
	for i, item := range items {
		uri := item.GetUri()
		if uri == "" || strings.HasPrefix(uri, "spotify:start-group:") || strings.HasPrefix(uri, "spotify:end-group:") {
			continue
		}
		if strings.Contains(uri, ":collection") {
			continue
		}
		id := playlistIDFromURI(uri)
		if id == "" {
			continue
		}
		info := playlist.PlaylistInfo{ID: id, Name: id, Section: "Followed playlists"}
		if i < len(metas) && metas[i] != nil {
			if attrs := metas[i].GetAttributes(); attrs != nil && attrs.GetName() != "" {
				info.Name = attrs.GetName()
			}
			info.TrackCount = int(metas[i].GetLength())
			if owner := metas[i].GetOwnerUsername(); owner != "" && userID != "" && owner == userID {
				info.Section = "Your playlists"
			}
		}
		out = append(out, info)
	}
	return out
}

func playlistIDFromURI(uri string) string {
	switch {
	case strings.HasPrefix(uri, "spotify:playlist:"):
		return strings.TrimPrefix(uri, "spotify:playlist:")
	case strings.Contains(uri, ":playlist:"):
		if i := strings.LastIndex(uri, ":playlist:"); i >= 0 {
			return uri[i+len(":playlist:"):]
		}
	}
	return ""
}

func (p *SpotifyProvider) fetchTracksSpclient(ctx context.Context, playlistID string) ([]playlist.Track, error) {
	uri := "spotify:playlist:" + playlistID
	if playlistID == "YOUR MUSIC" {
		user := p.currentUserID(ctx)
		if user == "" {
			return nil, fmt.Errorf("spotify: no username for liked songs")
		}
		uri = "spotify:user:" + user + ":collection"
	}
	ctxRes, err := p.session.contextResolve(ctx, uri)
	if err != nil {
		return nil, err
	}
	tracks := tracksFromContext(ctxRes)
	warnHydrate(p.hydrateTracks(ctx, tracks))
	return tracks, nil
}

func (s *Session) contextResolve(ctx context.Context, uri string) (*connectpb.Context, error) {
	if s == nil {
		return nil, fmt.Errorf("spotify: session closed")
	}
	s.mu.RLock()
	sess := s.sess
	s.mu.RUnlock()
	if sess == nil {
		return nil, fmt.Errorf("spotify: session closed")
	}
	return sess.Spclient().ContextResolve(ctx, uri)
}

func tracksFromContext(ctx *connectpb.Context) []playlist.Track {
	if ctx == nil {
		return nil
	}
	var out []playlist.Track
	for _, page := range ctx.GetPages() {
		for _, tr := range page.GetTracks() {
			uri := tr.GetUri()
			if uri == "" {
				continue
			}
			md := tr.GetMetadata()
			title := firstMeta(md, "title", "name")
			if title == "" {
				title = uri
			}
			dur := 0
			if ms := firstMeta(md, "duration", "duration_ms"); ms != "" {
				if n, err := strconv.Atoi(ms); err == nil {
					dur = n / 1000
					if n < 1000 && n > 0 {
						dur = n
					}
				}
			}
			out = append(out, playlist.Track{
				Path:         uri,
				Title:        title,
				Artist:       firstMeta(md, "artist_name", "artist"),
				Album:        firstMeta(md, "album_title", "album"),
				DurationSecs: dur,
				Stream:       false,
			})
		}
	}
	return out
}

func firstMeta(md map[string]string, keys ...string) string {
	if md == nil {
		return ""
	}
	for _, k := range keys {
		if v := md[k]; v != "" {
			return v
		}
	}
	return ""
}

func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "rate-limited") || strings.Contains(err.Error(), "429")
}

func logPlaylistFallback(err error) {
	applog.Warn("spotify: web api playlist fetch failed (%v); using librespot playlist protocol", err)
}
