package minihls

import (
	"encoding/hex"
	"errors"
	"net/http"
	netUrl "net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aobeom/minireq/v2"
	"github.com/aobeom/minitools"
)

type HLS struct {
	indexUrl string
	mainUrl  string
	key      string
	iv       string
	cookies  []*http.Cookie
}

var client = minireq.NewClient()

func (hls *HLS) userAgent() string {
	UA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.5060.134 Safari/537.36 Edg/103.0.1264.77"
	return UA
}

func (hls *HLS) genHost(typ string) (string, error) {
	switch typ {
	case "key":
		u, err := netUrl.Parse(hls.mainUrl)
		if err != nil {
			return "", err
		}
		host := u.Scheme + "://" + u.Host
		return host, nil
	case "yahoo":
		u1, err := netUrl.Parse(hls.indexUrl)
		if err != nil {
			return "", err
		}

		u2, err := netUrl.Parse(hls.mainUrl)
		if err != nil {
			return "", err
		}
		host := u1.Scheme + "://" + u1.Host + u1.Path + "?" + u2.RawQuery
		return host, nil
	default:
		return "", errors.New("type not found")
	}
}

func (hls *HLS) FilterHD(url string) (string, error) {
	header := minireq.Headers{"user-agent": hls.userAgent()}
	res, err := client.Get(url, header)
	if err != nil {
		return "", err
	}

	rawData, err := res.RawData()
	if err != nil {
		return "", err
	}

	playlistContent := string(rawData)
	resolutionRule := regexp.MustCompile(`(?sm)RESOLUTION=([\d]+x[\d]+)`)
	bandwidthRule := regexp.MustCompile(`(?sm)BANDWIDTH=([\d]+)`)
	bestRule := regexp.MustCompile(`(?m)^\w.*`)

	results := bandwidthRule.FindAllStringSubmatch(playlistContent, -1)
	if len(results) == 0 {
		results = resolutionRule.FindAllStringSubmatch(playlistContent, -1)
	}
	playlistURLs := bestRule.FindAllString(playlistContent, -1)

	maxCount := 0
	maxIndex := 0
	for index, result := range results {
		count, err := strconv.Atoi(result[1])
		if err != nil {
			return "", err
		}
		if count > maxCount {
			maxCount = count
			maxIndex = index
		}
	}

	playlistHD := playlistURLs[maxIndex]
	hls.indexUrl = url
	hls.mainUrl = playlistHD
	if !strings.HasPrefix(playlistHD, "http") {
		host, err := hls.genHost("yahoo")
		if err != nil {
			return "", err
		}
		playlistHD = host
		hls.mainUrl = playlistHD
	}
	return playlistHD, nil
}

func (hls *HLS) filterKey() ([]byte, error) {
	keyUrl := hls.key
	if keyUrl == "" {
		return nil, errors.New("key url is empty")
	}

	if !strings.HasPrefix(keyUrl, "http") {
		host, err := hls.genHost("key")
		if err != nil {
			return nil, err
		}
		keyUrl = host + keyUrl
	}

	header := minireq.Headers{"user-agent": hls.userAgent()}
	res, err := client.Get(keyUrl, header, hls.cookies)
	if err != nil {
		return nil, err
	}
	rawData, err := res.RawData()
	if err != nil {
		return nil, err
	}
	if len(rawData) != 16 {
		return nil, errors.New("key length is error: " + string(rawData))
	}
	return rawData, nil
}

func (hls *HLS) filterIV() ([]byte, error) {
	iv := hls.iv
	if iv == "" {
		return nil, nil
	}
	iv = strings.ReplaceAll(iv, "0x", "")
	ivByte, err := hex.DecodeString(iv)
	if err != nil {
		return nil, err
	}
	return ivByte, nil
}

func (hls *HLS) FilterVideo(url string) ([]string, error) {
	header := minireq.Headers{"user-agent": hls.userAgent()}
	res, err := client.Get(url, header, hls.cookies)
	if err != nil {
		return nil, err
	}
	rawData, err := res.RawData()
	if err != nil {
		return nil, err
	}
	hls.cookies = res.Response.Cookies()
	hdData := string(rawData)

	keyRule := regexp.MustCompile(`URI="(.*)"`)
	ivRule := regexp.MustCompile(`IV=(.*)`)

	keyData := keyRule.FindAllStringSubmatch(hdData, -1)
	if len(keyData) != 0 {
		hls.key = keyData[0][1]
	} else {
		hls.key = ""
	}

	ivData := ivRule.FindAllStringSubmatch(hdData, -1)
	if len(ivData) != 0 {
		hls.iv = ivData[0][1]
	} else {
		hls.iv = ""
	}

	videoRule := regexp.MustCompile(`(?m)^\w.*`)
	videos := videoRule.FindAllString(hdData, -1)

	return videos, nil
}

func (hls *HLS) Download(output string, videos []string) error {
	key, err := hls.filterKey()
	if err != nil {
		return err
	}
	iv, err := hls.filterIV()
	if err != nil {
		return err
	}

	videoFile, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer videoFile.Close()

	for _, video := range videos {
		header := minireq.Headers{"user-agent": hls.userAgent()}
		res, err := client.Get(video, header)
		if err != nil {
			return err
		}
		rawData, err := res.RawData()
		if err != nil {
			return err
		}
		decData, err := minitools.AESSuite().Decrypt(rawData, key, iv)
		if err != nil {
			return err
		}

		offset, err := videoFile.Seek(0, os.SEEK_END)
		if err != nil {
			return err
		}
		_, err = videoFile.WriteAt(decData, offset)
		if err != nil {
			return err
		}
	}
	return nil
}
