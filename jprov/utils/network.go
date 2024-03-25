package utils

import (
    "bytes"
    "net/http"
    "fmt"
    "io"
)

func TestDownloadFileFromURL(url string, fid string) (int64, error) {
	cli := http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/download/%s", url, fid), nil)
	if err != nil {
		return 0, err
	}

	req.Header = http.Header{
		"User-Agent":                {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.62 Safari/537.36"},
		"Upgrade-Insecure-Requests": {"1"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"},
		"Accept-Encoding":           {"gzip, deflate, br"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Connection":                {"keep-alive"},
	}

	resp, err := cli.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("failed to find file on network")
	}

	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, resp.Body)
	if err != nil {
		return 0, err
	}

	return size, nil
}
