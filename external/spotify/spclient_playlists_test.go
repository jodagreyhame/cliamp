package spotify

import (
	"testing"

	playlist4pb "github.com/devgianlu/go-librespot/proto/spotify/playlist4"
	"google.golang.org/protobuf/proto"
)

func TestPlaylistsFromRootlistSkipsFolders(t *testing.T) {
	name := "Owned Mix"
	length := int32(12)
	owner := "me"
	list := &playlist4pb.SelectedListContent{
		Contents: &playlist4pb.ListItems{
			Items: []*playlist4pb.Item{
				{Uri: proto.String("spotify:start-group:abc")},
				{Uri: proto.String("spotify:playlist:plowned")},
				{Uri: proto.String("spotify:end-group:abc")},
				{Uri: proto.String("spotify:user:other:playlist:plfollowed")},
				{Uri: proto.String("spotify:user:me:collection")},
			},
			MetaItems: []*playlist4pb.MetaItem{
				{},
				{Attributes: &playlist4pb.ListAttributes{Name: proto.String(name)}, Length: proto.Int32(length), OwnerUsername: proto.String(owner)},
				{},
				{Attributes: &playlist4pb.ListAttributes{Name: proto.String("Followed")}, Length: proto.Int32(3), OwnerUsername: proto.String("other")},
				{},
			},
		},
	}
	got := playlistsFromRootlist("me", list)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (library + 2 playlists): %#v", len(got), got)
	}
	if got[0].ID != "YOUR MUSIC" || got[0].Section != "Library" {
		t.Fatalf("first = %#v, want Your Music library", got[0])
	}
	if got[1].ID != "plowned" || got[1].Name != name || got[1].Section != "Your playlists" || got[1].TrackCount != 12 {
		t.Fatalf("owned = %#v", got[1])
	}
	if got[2].ID != "plfollowed" || got[2].Section != "Followed playlists" {
		t.Fatalf("followed = %#v", got[2])
	}
}
