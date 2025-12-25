package buildin

import (
	"agent/tools"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func NewGetCurrentTimeTool() tools.Tool {
	return tools.New(
		"get_current_time",
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
			loc, err := time.LoadLocation(input.Location)
			if err != nil {
				return "", fmt.Errorf("load location: %w", err)
			}
			return time.Now().In(loc).Format("Mon Jan 2 15:04:05 MST 2006"), nil
		},
		tools.WithDescription("Get the current local time for a specified location."),
		tools.WithParameters(tools.ObjectSchema(map[string]any{
			"location": tools.StringProperty("Provide a valid IANA time zone (e.g., 'America/New_York', 'Europe/London')."),
		}, "location")),
	)
}
