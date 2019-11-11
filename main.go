package main

import (
	"bufio"
	"fmt"
	"hlsvideo/aes"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

var proxyURL string
var playlistURL string
var myClient http.Client = httpClient(proxyURL)

type m3u8Info struct {
	keyURL    string
	videoURLs []string
}

func getMax(data []int64) (maxIndex int64) {
	maxValue := data[0]
	maxIndex = int64(0)
	for index, value := range data {
		if maxValue < value {
			maxValue = value
			maxIndex = int64(index)
		}
	}
	return maxIndex
}

func getBestPlayURL(playlist string) (url string) {
	regResRule := regexp.MustCompile(`RESOLUTION\=(\d+x\d+)`)
	resGroup := regResRule.FindAllStringSubmatch(playlist, -1)

	regURLRule := regexp.MustCompile(`(?m)^[\w\-\.\/\:\?\&\=\%\,\+]+`)
	urlGroup := regURLRule.FindAllString(playlist, -1)

	var resIntList []int64
	for _, res := range resGroup {
		resString := strings.Replace(res[1], "x", "", -1)
		resInt, _ := strconv.ParseInt(resString, 10, 64)
		resIntList = append(resIntList, resInt)
	}
	maxIndex := getMax(resIntList)
	url = urlGroup[maxIndex]
	return
}

func getM3u8Data(playlistURL string, m3u8Raw string) (m3u8Data *m3u8Info) {
	m3u8Data = new(m3u8Info)
	regKeyRule := regexp.MustCompile(`"(.*?)"`)
	keyRaw := regKeyRule.FindAllString(m3u8Raw, -1)
	keyurl := strings.Replace(keyRaw[0], "\"", "", -1)
	keyurl = getHost(playlistURL, "m3u8url") + keyurl

	regURLRule := regexp.MustCompile(`(?m)^[\w\-\.\/\:\?\&\=\%\,\+]+`)
	urls := regURLRule.FindAllString(m3u8Raw, -1)

	m3u8Data.keyURL = keyurl
	m3u8Data.videoURLs = urls
	return
}

func getHost(url string, mode string) (host string) {
	urlPart := strings.Split(url, "/")
	if mode == "besturl" {
		url4Best := urlPart[1 : len(urlPart)-1]
		host = "https:/" + strings.Join(url4Best, "/") + "/"
	} else if mode == "m3u8url" {
		url4m3u8 := urlPart[1:3]
		host = "https:/" + strings.Join(url4m3u8, "/") + "/"
	}
	return
}

func rProgress(i int, amp float64) {
	progress := float64(i) * amp
	num := int(progress / 10)
	if num < 1 {
		num = 1
	}
	pstyle := strings.Repeat(">", num)
	log.Printf("%s [%.2f %%]\r", pstyle, progress)
}

// DownloadVideo Download videos
func DownloadVideo(m3u8Data *m3u8Info, savePath string) {
	videoFile, _ := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	keyurl := m3u8Data.keyURL
	urls := m3u8Data.videoURLs
	total := float64(len(urls))
	part, _ := strconv.ParseFloat(fmt.Sprintf("%.5f", 100.0/total), 64)
	keyBytes := httpGet(keyurl)
	for i, url := range urls {
		i = i + 1
		videoBytes := httpGet(url)
		decrtVideo := aes.Decrypt(videoBytes, keyBytes)
		offset, _ := videoFile.Seek(0, os.SEEK_END)
		videoFile.WriteAt(decrtVideo, offset)
		rProgress(i, part)
	}
	log.Println("Finished [ " + savePath + " ]")
	defer videoFile.Close()
}

func s5Proxy(proxyURL string) (transport *http.Transport) {
	dialer, err := proxy.SOCKS5("tcp", proxyURL,
		nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		log.Fatal("Proxy Error")
	}
	transport = &http.Transport{
		Proxy:               nil,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return
}

func httpClient(proxy string) (client http.Client) {
	client = http.Client{Timeout: 30 * time.Second}
	if proxy != "" {
		transport := s5Proxy(proxy)
		client = http.Client{Timeout: 30 * time.Second, Transport: transport}
	}
	return
}

func httpGet(url string) []byte {
	UserAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.87 Safari/537.36"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", UserAgent)
	res, _ := myClient.Do(req)
	body, _ := ioutil.ReadAll(res.Body)
	return body
}

func paras() {
	fmt.Print("Playlist URL: ")
	url := bufio.NewScanner(os.Stdin)
	url.Scan()
	fmt.Print("Use Proxy (Default \"\"): ")
	proxy := bufio.NewScanner(os.Stdin)
	proxy.Scan()
	playlistURL = url.Text()
	proxyURL = proxy.Text()
	myClient = httpClient(proxyURL)
}

func main() {
	// playlistURL := "https://gw-yvpub.c.yimg.jp/v1/hls/ZW7t9XSN3GOLSw8I/video.m3u8?min_bw=250&https=1"
	paras()
	// Best playlist URL
	log.Println("Analysis URL...")
	playlist := httpGet(playlistURL)
	bestURL := getBestPlayURL(string(playlist))
	bestURL = getHost(playlistURL, "besturl") + bestURL

	// Key and Video Urls
	log.Println("Get the url of the video and key...")
	m3u8RawData := httpGet(bestURL)
	m3u8Data := getM3u8Data(playlistURL, string(m3u8RawData))
	timeStr := time.Now().Format("20060102150405")
	savePath := "all_" + timeStr + ".ts"
	log.Println("Downloading...")
	DownloadVideo(m3u8Data, savePath)
}
