package gmgn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"net/url"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

func scraperDoRequest(ctx context.Context, client *http.Client, apiKey string, method, targetUrl string, bodyJson any) (string, error) {
	var body io.Reader
	if bodyJson != nil {
		data, err := json.Marshal(bodyJson)
		if err != nil {
			return "", err
		}
		body = bytes.NewBuffer(data)
	}

	fullUrl := fmt.Sprintf("https://api.zenrows.com/v1/?apikey=%s&url=%s&response_type=plaintext", apiKey, url.QueryEscape(targetUrl))
	req, err := http.NewRequestWithContext(ctx, method, fullUrl, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("accept-encoding", "gzip,deflate,br")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	content := res.Header["Content-Type"]
	encoding := res.Header["Content-Encoding"]
	resBody := cycletls.DecompressBody(bodyBytes, encoding, content)

	if res.StatusCode > 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("status: %d", res.StatusCode)
	}

	return string(resBody), nil
}
