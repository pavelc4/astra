package capcut

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

func FetchData(ctx context.Context, targetURL string) (*Result, error) {
	payload, _ := json.Marshal(map[string]string{"url": targetURL})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, GenViralEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", RefererURL)
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Capcut API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("Capcut response read failed")
	}

	return &Result{Platform: PlatformName, Raw: json.RawMessage(body)}, nil
}
