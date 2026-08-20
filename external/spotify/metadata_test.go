package spotify

import (
	"strings"
	"testing"

	"github.com/bjarneo/cliamp/playlist"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestNeedsHydration(t *testing.T) {
	uri := "spotify:track:abc"
	cases := []struct {
		t    playlist.Track
		want bool
	}{
		{playlist.Track{Path: uri, Title: "Real Song"}, false},
		{playlist.Track{Path: uri, Title: uri}, true},
		{playlist.Track{Path: uri, Title: ""}, true},
		{playlist.Track{Path: uri, Title: "spotify:track:other"}, true},
	}
	for _, tc := range cases {
		if got := needsHydration(tc.t); got != tc.want {
			t.Errorf("needsHydration(%q) = %v, want %v", tc.t.Title, got, tc.want)
		}
	}
}

func TestNormalizeSpotifyTracksClearsStream(t *testing.T) {
	in := []playlist.Track{
		{Path: "spotify:track:a", Title: "A", Stream: true},
		{Path: "spotify:track:b", Title: "B", Stream: false},
	}
	got, changed := normalizeSpotifyTracks(in)
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if !in[0].Stream {
		t.Fatal("normalize mutated the input slice")
	}
	if got[0].Stream || got[1].Stream {
		t.Fatalf("Stream flags = %#v", got)
	}
}

func TestApplyTrackProto(t *testing.T) {
	year := int32(1999)
	dur := int32(181000)
	num := int32(4)
	src := &metadatapb.Track{
		Name:     proto.String("Around the World"),
		Duration: &dur,
		Number:   &num,
		Artist:   []*metadatapb.Artist{{Name: proto.String("Daft Punk")}},
		Album:    &metadatapb.Album{Name: proto.String("Discovery"), Date: &metadatapb.Date{Year: &year}},
	}
	dst := playlist.Track{Path: "spotify:track:abc", Title: "spotify:track:abc"}
	applyTrackProto(&dst, src)
	if dst.Title != "Around the World" || dst.Artist != "Daft Punk" || dst.Album != "Discovery" {
		t.Fatalf("meta = %#v", dst)
	}
	if dst.Year != 1999 || dst.DurationSecs != 181 || dst.TrackNumber != 4 {
		t.Fatalf("year/dur/num = %#v", dst)
	}
}

func TestApplyMetadataResponseHydratesTitles(t *testing.T) {
	uri := "spotify:track:abc"
	payload, err := anypb.New(&metadatapb.Track{
		Name:   proto.String("Harder Better"),
		Artist: []*metadatapb.Artist{{Name: proto.String("Daft Punk")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	tracks := []playlist.Track{{Path: uri, Title: uri, Stream: false}}
	idx := map[string][]int{uri: {0}}
	applyMetadataResponse(tracks, idx, &extmetadatapb.BatchedExtensionResponse{
		ExtendedMetadata: []*extmetadatapb.EntityExtensionDataArray{{
			ExtensionKind: extmetadatapb.ExtensionKind_TRACK_V4,
			ExtensionData: []*extmetadatapb.EntityExtensionData{{
				EntityUri:     uri,
				Header:        &extmetadatapb.EntityExtensionDataHeader{StatusCode: 200},
				ExtensionData: payload,
			}},
		}},
	})
	if tracks[0].Title != "Harder Better" || tracks[0].Artist != "Daft Punk" {
		t.Fatalf("hydrated = %#v", tracks[0])
	}
}

func TestTracksFromContextUsesLibrespotPlayback(t *testing.T) {
	uri := "spotify:track:abc"
	ctx := &connectpb.Context{
		Pages: []*connectpb.ContextPage{{
			Tracks: []*connectpb.ContextTrack{
				{Uri: uri},
				{Uri: "spotify:track:named", Metadata: map[string]string{"title": "Named", "artist_name": "Someone", "duration": "120000"}},
			},
		}},
	}
	got := tracksFromContext(ctx)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Path != uri || got[0].Title != uri || got[0].Stream {
		t.Fatalf("empty meta = %#v, want URI title and Stream false", got[0])
	}
	if got[1].Title != "Named" || got[1].Artist != "Someone" || got[1].DurationSecs != 120 || got[1].Stream {
		t.Fatalf("named = %#v", got[1])
	}
}

func TestServeTracksClearsStreamWithoutNetwork(t *testing.T) {
	dir := t.TempDir()
	p := New(nil, "client", 320)
	p.cacheDir = dir
	uri := "spotify:track:abc"
	got := p.serveTracks("pl1", "snap", []playlist.Track{{Path: uri, Title: uri, Stream: true}})
	if len(got) != 1 || got[0].Stream {
		t.Fatalf("got = %#v, want Stream false", got)
	}
	if !strings.HasPrefix(got[0].Title, "spotify:") {
		t.Fatalf("title = %q, want URI until metadata hydrate", got[0].Title)
	}
	loaded, snap, ok := p.loadTracks("pl1")
	if !ok || snap != "snap" || loaded[0].Stream {
		t.Fatalf("disk = %#v snap=%q ok=%v, want persisted Stream false", loaded, snap, ok)
	}
}
