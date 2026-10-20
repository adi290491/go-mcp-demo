package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxForecastPeriods = 5

func getForecast(ctx context.Context, req *mcp.CallToolRequest, input ForecastInput) (*mcp.CallToolResult, any, error) {
	// NWS needs two hops: lat/long -> grid point -> forecast URL.
	pointsURL := fmt.Sprintf("%s/points/%f,%f", nwsAPIBase, input.Latitude, input.Longitude)
	points, err := makeNWSRequest[PointsResponse](ctx, pointsURL)
	if err != nil {
		return errorResult("Unable to fetch forecast data for this location."), nil, nil
	}

	forecastURL := points.Properties.Forecast
	if forecastURL == "" {
		return errorResult("Unable to fetch forecast URL."), nil, nil
	}

	forecast, err := makeNWSRequest[ForecastResponse](ctx, forecastURL)
	if err != nil {
		return errorResult("Unable to fetch detailed forecast."), nil, nil
	}

	periods := forecast.Properties.Periods
	if len(periods) == 0 {
		return textResult("No forecast periods available."), nil, nil
	}

	var formatted []string
	for _, p := range periods[:min(maxForecastPeriods, len(periods))] {
		formatted = append(formatted, formatPeriod(p))
	}
	return textResult(strings.Join(formatted, "\n---\n")), nil, nil
}

func getAlerts(ctx context.Context, req *mcp.CallToolRequest, input AlertsInput) (*mcp.CallToolResult, any, error) {
	alertsURL := fmt.Sprintf("%s/alerts/active/area/%s", nwsAPIBase, strings.ToUpper(input.State))
	alerts, err := makeNWSRequest[AlertsResponse](ctx, alertsURL)
	if err != nil {
		return errorResult("Unable to fetch alerts."), nil, nil
	}

	if len(alerts.Features) == 0 {
		return textResult("No active alerts for this state."), nil, nil
	}

	var formatted []string
	for _, f := range alerts.Features {
		formatted = append(formatted, formatAlert(f))
	}
	return textResult(strings.Join(formatted, "\n---\n")), nil, nil
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

// errorResult flags the result as a tool error so the client (and Claude)
// can tell a failed lookup apart from a successful one.
func errorResult(text string) *mcp.CallToolResult {
	res := textResult(text)
	res.IsError = true
	return res
}
