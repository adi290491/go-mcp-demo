package main

type PointsResponse struct {
	Properties struct {
		Forecast string `json:"forecast"`
	} `json:"properties"`
}

type ForecastResponse struct {
	Properties struct {
		Periods []ForecastPeriod `json:"periods"`
	} `json:"properties"`
}

type ForecastPeriod struct {
	Name             string `json:"name"`
	Temperature      int    `json:"temperature"`
	TemperatureUnit  string `json:"temperatureUnit"`
	WindSpeed        string `json:"windSpeed"`
	WindDirection    string `json:"windDirection"`
	DetailedForecast string `json:"detailedForecast"`
}

type AlertsResponse struct {
	Features []AlertFeature `json:"features"`
}

type AlertFeature struct {
	Properties AlertProperties `json:"properties"`
}

type AlertProperties struct {
	Event       string `json:"event"`
	AreaDesc    string `json:"areaDesc"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Instruction string `json:"instruction"`
}

type ForecastInput struct {
	Latitude  float64 `json:"latitude" jsonschema:"Latitude of the location"`
	Longitude float64 `json:"longitude" jsonschema:"Longitude of the location"`
}

type AlertsInput struct {
	State string `json:"state" jsonschema:"Two-letter US state code (e.g. CA, NY)"`
}
