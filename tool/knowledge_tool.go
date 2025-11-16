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

// WikipediaSearch performs a search on Wikipedia for the given query and returns a summary.
// It uses the Wikipedia API.
func WikipediaSearch(query string) (string, error) {
	baseURL := "https://en.wikipedia.org/w/api.php"
	params := url.Values{}
	params.Add("action", "query")
	params.Add("format", "json")
	params.Add("prop", "extracts")
	params.Add("exintro", "") // Return only content before the first section
	params.Add("explaintext", "") // Return plain text
	params.Add("redirects", "1") // Resolve redirects
	params.Add("titles", query)

	searchURL := baseURL + "?" + params.Encode()

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform Wikipedia search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Wikipedia API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result struct {
		Query struct {
			Pages map[string]struct {
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal Wikipedia response: %w", err)
	}

	for _, page := range result.Query.Pages {
		if page.Extract != "" {
			// Clean up some common Wikipedia API artifacts
			extract := strings.ReplaceAll(page.Extract, "(listen)", "")
			extract = strings.TrimSpace(extract)
			return extract, nil
		}
	}

	return "No relevant Wikipedia entry found.", nil
}