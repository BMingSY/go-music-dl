package library

import (
	"context"
	"github.com/guohuiyuan/music-lib/model"
	"testing"
)

type multipartProvider struct {
	Provider
	collectionProvider
}

func (multipartProvider) ParsePlaylist(string) (*model.Playlist, []model.Song, error) {
	return &model.Playlist{ID: "season:1:2:BVfixture", Source: "bilibili", Name: "测试合集", Extra: map[string]string{"type": "season"}}, []model.Song{{ID: "one|1", Source: "bilibili", Name: "第一首"}, {ID: "two|2", Source: "bilibili", Name: "第二首"}}, nil
}
func TestVideoLinkKeepsCollectionAndAllTracks(t *testing.T) {
	r, e := searchRemote(context.Background(), "bilibili", "https://www.bilibili.com/video/BV1YmHA6WEJz/", multipartProvider{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Container == nil || r.Container.Name != "测试合集" || len(r.Tracks) != 2 {
		t.Fatalf("lost collection: %+v", r)
	}
}
