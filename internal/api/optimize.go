package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const defaultOptimizeBaseURL = "https://optimize.rime.ai"
const EnvOptimizeURL = "RIME_OPTIMIZE_URL"

type OptimizeClient struct {
	baseURL          string
	apiKey           string
	authHeaderPrefix string
	userAgent        string
	client           *http.Client
}

func NewOptimizeClient(apiKey string, version string) *OptimizeClient {
	baseURL := defaultOptimizeBaseURL
	if url := os.Getenv(EnvOptimizeURL); url != "" {
		baseURL = url
	}
	return &OptimizeClient{
		baseURL:          baseURL,
		apiKey:           apiKey,
		authHeaderPrefix: "Bearer",
		userAgent:        UserAgent(version),
		client:           &http.Client{},
	}
}

type UsageDay struct {
	Day         string `json:"day"`
	MistChars   int64  `json:"mistChars"`
	ArcanaChars int64  `json:"arcanaChars"`
}

type UsageHistory struct {
	Data []UsageDay `json:"data"`
}

func (c *OptimizeClient) GetRecentUsage() (*UsageHistory, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/usage/recent-history", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.authHeaderPrefix, c.apiKey))
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("authentication failed: invalid API key")
		default:
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}
	}

	var history UsageHistory
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &history, nil
}
