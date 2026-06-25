package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Sample struct {
	Timestamp time.Time
	Value     float64
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type queryResult struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]Sample, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("start", fmt.Sprintf("%d", start.Unix()))
	q.Set("end", fmt.Sprintf("%d", end.Unix()))
	q.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	const maxBodySize = 10 << 20 // 10MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var qr queryResult
	if err := json.Unmarshal(body, &qr); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if qr.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", string(body))
	}

	var samples []Sample
	for _, r := range qr.Data.Result {
		for _, v := range r.Values {
			if len(v) == 2 {
				ts := time.Unix(int64(v[0].(float64)), 0)
				val, _ := v[1].(float64)
				samples = append(samples, Sample{Timestamp: ts, Value: val})
			}
		}
	}
	return samples, nil
}
