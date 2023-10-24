package minihls

import (
	"testing"
)

func TestHLS(t *testing.T) {
	hls := new(HLS)

	url := "https://gw-yvpub.yahoo.co.jp/v1/hls/74a87649b78de1762be677cd3334d5c4/video.m3u8?min_bw=250&https=1"
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
