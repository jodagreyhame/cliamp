package spotify

import (
	"context"
	"fmt"
	"strings"

	"github.com/bjarneo/cliamp/applog"
	"github.com/bjarneo/cliamp/playlist"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
)

const metadataBatchSize = 50

func needsHydration(t playlist.Track) bool {
	if t.Title == "" || t.Title == t.Path {
		return true
	}
	return strings.HasPrefix(t.Title, "spotify:")
}

func tracksNeedHydration(tracks []playlist.Track) bool {
	for _, t := range tracks {
		if needsHydration(t) {
			return true
		}
	}
	return false
}

func trackMetaChanged(a, b []playlist.Track) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Artist != b[i].Artist || a[i].Album != b[i].Album || a[i].DurationSecs != b[i].DurationSecs {
			return true
		}
	}
	return false
}

func normalizeSpotifyTracks(tracks []playlist.Track) ([]playlist.Track, bool) {
	changed := false
	for i := range tracks {
		if !tracks[i].Stream {
			continue
		}
		if !changed {
			tracks = append([]playlist.Track(nil), tracks...)
			changed = true
		}
		tracks[i].Stream = false
	}
	return tracks, changed
}

func applyTrackProto(dst *playlist.Track, src *metadatapb.Track) {
	if src == nil || dst == nil {
		return
	}
	if name := src.GetName(); name != "" {
		dst.Title = name
	}
	if arts := src.GetArtist(); len(arts) > 0 {
		names := make([]string, 0, len(arts))
		for _, a := range arts {
			if n := a.GetName(); n != "" {
				names = append(names, n)
			}
		}
		if len(names) > 0 {
			dst.Artist = strings.Join(names, ", ")
		}
	}
	if alb := src.GetAlbum(); alb != nil {
		if n := alb.GetName(); n != "" {
			dst.Album = n
		}
		if d := alb.GetDate(); d != nil && d.GetYear() > 0 {
			dst.Year = int(d.GetYear())
		}
	}
	if ms := src.GetDuration(); ms > 0 {
		dst.DurationSecs = int(ms) / 1000
	}
	if n := src.GetNumber(); n > 0 {
		dst.TrackNumber = int(n)
	}
}

func applyEpisodeProto(dst *playlist.Track, src *metadatapb.Episode) {
	if src == nil || dst == nil {
		return
	}
	if name := src.GetName(); name != "" {
		dst.Title = name
	}
	if show := src.GetShow(); show != nil {
		if n := show.GetName(); n != "" {
			dst.Artist = n
			dst.Album = n
		}
	}
	if ms := src.GetDuration(); ms > 0 {
		dst.DurationSecs = int(ms) / 1000
	}
}

func applyMetadataResponse(tracks []playlist.Track, idx map[string][]int, resp *extmetadatapb.BatchedExtensionResponse) {
	if resp == nil {
		return
	}
	for _, item := range resp.GetExtendedMetadata() {
		kind := item.GetExtensionKind()
		for _, ext := range item.GetExtensionData() {
			if hdr := ext.GetHeader(); hdr != nil && hdr.GetStatusCode() != 0 && hdr.GetStatusCode() != 200 {
				continue
			}
			payload := ext.GetExtensionData()
			if payload == nil {
				continue
			}
			uri := ext.GetEntityUri()
			indexes := idx[uri]
			if len(indexes) == 0 {
				continue
			}
			switch kind {
			case extmetadatapb.ExtensionKind_TRACK_V4:
				var meta metadatapb.Track
				if err := payload.UnmarshalTo(&meta); err != nil {
					continue
				}
				for _, i := range indexes {
					applyTrackProto(&tracks[i], &meta)
				}
			case extmetadatapb.ExtensionKind_EPISODE_V4:
				var meta metadatapb.Episode
				if err := payload.UnmarshalTo(&meta); err != nil {
					continue
				}
				for _, i := range indexes {
					applyEpisodeProto(&tracks[i], &meta)
				}
			}
		}
	}
}

func (p *SpotifyProvider) hydrateTracks(ctx context.Context, tracks []playlist.Track) error {
	if p == nil || p.session == nil {
		return nil
	}
	var trackURIs, episodeURIs []string
	idx := make(map[string][]int)
	for i, t := range tracks {
		if !needsHydration(t) {
			continue
		}
		uri := t.Path
		switch {
		case strings.HasPrefix(uri, "spotify:track:"):
			if _, ok := idx[uri]; !ok {
				trackURIs = append(trackURIs, uri)
			}
		case strings.HasPrefix(uri, "spotify:episode:"):
			if _, ok := idx[uri]; !ok {
				episodeURIs = append(episodeURIs, uri)
			}
		default:
			continue
		}
		idx[uri] = append(idx[uri], i)
	}
	if err := p.hydrateURIBatch(ctx, trackURIs, idx, tracks, extmetadatapb.ExtensionKind_TRACK_V4); err != nil {
		return err
	}
	return p.hydrateURIBatch(ctx, episodeURIs, idx, tracks, extmetadatapb.ExtensionKind_EPISODE_V4)
}

func (p *SpotifyProvider) hydrateURIBatch(ctx context.Context, uris []string, idx map[string][]int, tracks []playlist.Track, kind extmetadatapb.ExtensionKind) error {
	for i := 0; i < len(uris); i += metadataBatchSize {
		end := min(i+metadataBatchSize, len(uris))
		req := &extmetadatapb.BatchedEntityRequest{
			EntityRequest: entityRequests(uris[i:end], kind),
		}
		resp, err := p.session.extendedMetadata(ctx, req)
		if err != nil {
			return fmt.Errorf("spotify: extended metadata: %w", err)
		}
		applyMetadataResponse(tracks, idx, resp)
	}
	return nil
}

func entityRequests(uris []string, kind extmetadatapb.ExtensionKind) []*extmetadatapb.EntityRequest {
	out := make([]*extmetadatapb.EntityRequest, 0, len(uris))
	for _, uri := range uris {
		out = append(out, &extmetadatapb.EntityRequest{
			EntityUri: uri,
			Query:     []*extmetadatapb.ExtensionQuery{{ExtensionKind: kind}},
		})
	}
	return out
}

func (s *Session) extendedMetadata(ctx context.Context, req *extmetadatapb.BatchedEntityRequest) (*extmetadatapb.BatchedExtensionResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("spotify: session closed")
	}
	s.mu.RLock()
	sess := s.sess
	s.mu.RUnlock()
	if sess == nil {
		return nil, fmt.Errorf("spotify: session closed")
	}
	return sess.Spclient().ExtendedMetadata(ctx, req)
}

func warnHydrate(err error) {
	if err != nil {
		applog.Warn("spotify: track metadata hydrate: %v", err)
	}
}
