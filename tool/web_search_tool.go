package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DuckDuckGoSearch performs a DuckDuckGo search for the given query.
// It uses the DuckDuckGo Instant Answer API.
func DuckDuckGoSearch(query string) (string, error) {
	baseURL := "https://api.duckduckgo.com/?format=json&q="
	searchURL := baseURL + url.QueryEscape(query)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform DuckDuckGo search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DuckDuckGo API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			URL  string `json:"URL"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal DuckDuckGo response: %w", err)
	}

	if result.AbstractText != "" {
		return fmt.Sprintf("%s (Source: %s)", result.AbstractText, result.AbstractURL), nil
	} else if len(result.RelatedTopics) > 0 {
		// Fallback to related topics if no abstract
		var topics []string
		for _, topic := range result.RelatedTopics {
			topics = append(topics, topic.Text)
		}
		return fmt.Sprintf("No direct abstract found. Related topics: %s", strings.Join(topics, "; ")), nil
	}

	return "No relevant information found.", nil
}

// SerpAPISearch is removed as it requires an API key and is complex to implement directly.
// MetaphorSearch is removed as it requires an API key and is complex to implement directly.
// ScrapeURL is removed as it requires external libraries for robust scraping.
