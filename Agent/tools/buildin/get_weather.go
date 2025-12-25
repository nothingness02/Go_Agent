package buildin

import (
	"agent/tools"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func NewGetWeatherTool() tools.Tool {
	return tools.New(
		"get_weather",
		func(ctx context.Context, args string) (string, error) {
			var input struct {
				Location string `json:"location"`
			}
			if err := json.Unmarshal([]byte(args), &input); err != nil {
				return "", fmt.Errorf("parse args: %w", err)
			}
			if strings.TrimSpace(input.Location) == "" {
				return "", fmt.Errorf("location is required")
			}

			weatherURL := "https://wttr.in/" + url.PathEscape(input.Location) + "?format=3"
			client := &http.Client{Timeout: 10 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, weatherURL, nil)
			if err != nil {
				return "", fmt.Errorf("build request: %w", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("weather request: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("weather http status: %s", resp.Status)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("read response: %w", err)
			}
			return string(body), nil
		},
		tools.WithDescription("Get current weather for a location."),
		tools.WithParameters(tools.ObjectSchema(map[string]any{
			"location": tools.StringProperty("City or location name."),
		}, "location")),
	)
}
