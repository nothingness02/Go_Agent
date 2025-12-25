package buildin

import (
	"agent/tools"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func NewWebSearchTool() tools.Tool {
	return tools.New(
		"web_search",
		func(ctx context.Context, args string) (string, error) {
			var input struct {
				Query      string `json:"query"`
				MaxResults int    `json:"max_results"`
			}
			if err := json.Unmarshal([]byte(args), &input); err != nil {
				return "", fmt.Errorf("parse args: %w", err)
			}
			if strings.TrimSpace(input.Query) == "" {
				return "", fmt.Errorf("query is required")
			}
			if input.MaxResults <= 0 {
				input.MaxResults = 5
			}

			searchURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(input.Query) + "&format=json&no_redirect=1&no_html=1"
			client := &http.Client{Timeout: 10 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
			if err != nil {
				return "", fmt.Errorf("build request: %w", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("search request: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("search http status: %s", resp.Status)
			}

			var data struct {
				Heading       string `json:"Heading"`
				AbstractText  string `json:"AbstractText"`
				RelatedTopics []struct {
					Text     string `json:"Text"`
					FirstURL string `json:"FirstURL"`
				} `json:"RelatedTopics"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return "", fmt.Errorf("decode response: %w", err)
			}

			lines := make([]string, 0, input.MaxResults+2)
			if data.Heading != "" {
				lines = append(lines, "Heading: "+data.Heading)
			}
			if data.AbstractText != "" {
				lines = append(lines, "Abstract: "+data.AbstractText)
			}
			count := 0
			for _, item := range data.RelatedTopics {
				if item.Text == "" {
					continue
				}
				lines = append(lines, fmt.Sprintf("- %s (%s)", item.Text, item.FirstURL))
				count++
				if count >= input.MaxResults {
					break
				}
			}
			if len(lines) == 0 {
				return "No instant answer found.", nil
			}
			return strings.Join(lines, "\n"), nil
		},
		tools.WithDescription("Search the web for information."),
		tools.WithParameters(tools.ObjectSchema(map[string]any{
			"query":       tools.StringProperty("The search query."),
			"max_results": tools.IntProperty("Maximum number of results to return."),
		}, "query")),
	)
}
