package prometheus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(queryResult{
			Status: "success",
			Data: struct {
				Result []struct {
					Values [][]interface{} `json:"values"`
				} `json:"result"`
			}{
				Result: []struct {
					Values [][]interface{} `json:"values"`
				}{
					{Values: [][]interface{}{{float64(1719000000), 0.5}, {float64(1719000060), 0.6}}},
				},
			},
		})
	}))
	defer server.Close()

	c := New(server.URL)
	samples, err := c.QueryRange(context.Background(), "test_metric", time.Now().Add(-1*time.Hour), time.Now(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
	if samples[0].Value != 0.5 {
		t.Errorf("expected 0.5, got %f", samples[0].Value)
	}
}
