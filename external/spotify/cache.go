package spotify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjarneo/cliamp/applog"
	"github.com/bjarneo/cliamp/internal/appdir"
	"github.com/bjarneo/cliamp/internal/fileutil"
	"github.com/bjarneo/cliamp/playlist"
)

const spotifyCacheSubdir = "spotify-cache"

type diskIndex struct {
	UserID    string         `json:"user_id"`
	SavedAt   time.Time      `json:"saved_at"`
	Playlists []diskPlaylist `json:"playlists"`
}

type diskPlaylist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TrackCount int    `json:"track_count"`
	Section    string `json:"section"`
	SnapshotID string `json:"snapshot_id,omitempty"`
}

type diskTracks struct {
	ID         string           `json:"id"`
	SnapshotID string           `json:"snapshot_id,omitempty"`
	Tracks     []playlist.Track `json:"tracks"`
}

func (p *SpotifyProvider) resolvedCacheDir() string {
	if p.cacheDir != "" {
		return p.cacheDir
	}
	dir, err := appdir.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, spotifyCacheSubdir)
}

func (p *SpotifyProvider) indexPath() string {
	dir := p.resolvedCacheDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "index.json")
}

func (p *SpotifyProvider) tracksPath(playlistID string) string {
	dir := p.resolvedCacheDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "tracks", cacheFileName(playlistID)+".json")
}

func cacheFileName(id string) string {
	if id == "YOUR MUSIC" {
		return "liked"
	}
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "playlist"
	}
	return b.String()
}

func (p *SpotifyProvider) loadIndex() (diskIndex, bool) {
	path := p.indexPath()
	if path == "" {
		return diskIndex{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return diskIndex{}, false
	}
	var idx diskIndex
	if err := json.Unmarshal(data, &idx); err != nil || len(idx.Playlists) == 0 {
		return diskIndex{}, false
	}
	return idx, true
}

func (p *SpotifyProvider) saveIndex(userID string, lists []playlist.PlaylistInfo) error {
	path := p.indexPath()
	if path == "" {
		return nil
	}
	idx := diskIndex{UserID: userID, SavedAt: time.Now(), Playlists: make([]diskPlaylist, 0, len(lists))}
	p.mu.Lock()
	for _, info := range lists {
		entry := diskPlaylist{
			ID:         info.ID,
			Name:       info.Name,
			TrackCount: info.TrackCount,
			Section:    info.Section,
		}
		if cached, ok := p.trackCache[info.ID]; ok {
			entry.SnapshotID = cached.snapshotID
		}
		idx.Playlists = append(idx.Playlists, entry)
	}
	p.mu.Unlock()
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func (p *SpotifyProvider) applyIndex(idx diskIndex) []playlist.PlaylistInfo {
	lists := make([]playlist.PlaylistInfo, 0, len(idx.Playlists))
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.userID == "" && idx.UserID != "" {
		p.userID = idx.UserID
		p.meFetched = true
	}
	for _, item := range idx.Playlists {
		lists = append(lists, playlist.PlaylistInfo{
			ID:         item.ID,
			Name:       item.Name,
			TrackCount: item.TrackCount,
			Section:    item.Section,
		})
		if item.SnapshotID == "" {
			continue
		}
		if cached, ok := p.trackCache[item.ID]; ok {
			if cached.snapshotID != item.SnapshotID {
				cached.tracks = nil
			}
			cached.snapshotID = item.SnapshotID
			continue
		}
		p.trackCache[item.ID] = &playlistCache{snapshotID: item.SnapshotID}
	}
	p.listCache = lists
	p.listCacheAt = time.Now()
	return lists
}

func (p *SpotifyProvider) loadTracks(playlistID string) ([]playlist.Track, string, bool) {
	path := p.tracksPath(playlistID)
	if path == "" {
		return nil, "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false
	}
	var stored diskTracks
	if err := json.Unmarshal(data, &stored); err != nil || len(stored.Tracks) == 0 {
		return nil, "", false
	}
	return stored.Tracks, stored.SnapshotID, true
}

func (p *SpotifyProvider) saveTracks(playlistID, snapshotID string, tracks []playlist.Track) error {
	path := p.tracksPath(playlistID)
	if path == "" {
		return nil
	}
	data, err := json.Marshal(diskTracks{ID: playlistID, SnapshotID: snapshotID, Tracks: tracks})
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func (p *SpotifyProvider) removeTracks(playlistID string) {
	if path := p.tracksPath(playlistID); path != "" {
		_ = os.Remove(path)
	}
}

func warnCache(op string, err error) {
	if err != nil {
		applog.Warn("spotify: %s cache: %v", op, err)
	}
}
