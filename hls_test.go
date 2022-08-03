package minihls

import (
	"testing"
)

func TestHLS(t *testing.T) {
	hls := new(HLS)

	url := "https://gw-yvpub.yahoo.co.jp/v1/hls/32fa6da1996c59bbee10d79e993d7d16/video.m3u8?min_bw=250&https=1"
	hdUrl, err := hls.FilterHD(url)
	if err != nil {
		t.Fatal(err)
	}
	videos, err := hls.FilterVideo(hdUrl)
	if err != nil {
		t.Fatal(err)
	}
	err = hls.Download("1.ts", videos)
	if err != nil {
		t.Fatal(err)
	}
}
