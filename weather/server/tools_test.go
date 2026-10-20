package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeNWS serves canned responses keyed by path and points nwsAPIBase at it.
func fakeNWS(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, strings.ReplaceAll(body, "{{base}}", "http://"+r.Host))
	}))
	t.Cleanup(srv.Close)

	old := nwsAPIBase
	nwsAPIBase = srv.URL
	t.Cleanup(func() { nwsAPIBase = old })
	return srv
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("want 1 content block, got %d", len(res.Content))
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestGetForecast(t *testing.T) {
	var periods []string
	for i := range 7 {
		periods = append(periods, fmt.Sprintf(`{"name":"Period%d","temperature":%d,"temperatureUnit":"F",
			"windSpeed":"5 mph","windDirection":"SW","detailedForecast":"Cloudy"}`, i, 50+i))
	}
	fakeNWS(t, map[string]string{
		"/points/37.774900,-122.419400": `{"properties":{"forecast":"{{base}}/gridpoints/MTR/1,2/forecast"}}`,
		"/gridpoints/MTR/1,2/forecast":  `{"properties":{"periods":[` + strings.Join(periods, ",") + `]}}`,
	})

	res, _, err := getForecast(context.Background(), nil, ForecastInput{Latitude: 37.7749, Longitude: -122.4194})
	if err != nil || res.IsError {
		t.Fatalf("unexpected failure: err=%v res=%+v", err, res)
	}

	text := resultText(t, res)
	if !strings.Contains(text, "Period0:") || !strings.Contains(text, "Temperature: 50°F") {
		t.Errorf("forecast missing first period:\n%s", text)
	}
	if !strings.Contains(text, "Period4:") || strings.Contains(text, "Period5:") {
		t.Errorf("forecast should contain exactly %d periods:\n%s", maxForecastPeriods, text)
	}
}

func TestGetForecastUnknownLocation(t *testing.T) {
	fakeNWS(t, map[string]string{})

	res, _, err := getForecast(context.Background(), nil, ForecastInput{Latitude: 0, Longitude: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Errorf("want IsError for failed lookup, got %q", resultText(t, res))
	}
}

func TestGetAlerts(t *testing.T) {
	fakeNWS(t, map[string]string{
		"/alerts/active/area/CA": `{"features":[
			{"properties":{"event":"Rip Current Statement","areaDesc":"Malibu","severity":"Moderate"}},
			{"properties":{}}]}`,
		"/alerts/active/area/NY": `{"features":[]}`,
	})

	res, _, err := getAlerts(context.Background(), nil, AlertsInput{State: "ca"})
	if err != nil || res.IsError {
		t.Fatalf("unexpected failure: err=%v res=%+v", err, res)
	}
	text := resultText(t, res)
	for _, want := range []string{
		"Event: Rip Current Statement", "Area: Malibu", "Severity: Moderate",
		"Event: Unknown", "No description available", // defaults for the empty alert
	} {
		if !strings.Contains(text, want) {
			t.Errorf("alerts missing %q:\n%s", want, text)
		}
	}

	res, _, _ = getAlerts(context.Background(), nil, AlertsInput{State: "NY"})
	if res.IsError || resultText(t, res) != "No active alerts for this state." {
		t.Errorf("unexpected empty-alerts result: %+v", res)
	}
}
