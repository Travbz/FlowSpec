package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLocation   = "Montrose"
	defaultState      = "Colorado"
	defaultCountry    = "US"
	defaultDays       = 5
	defaultGeoBaseURL = "https://geocoding-api.open-meteo.com"
	defaultWxBaseURL  = "https://api.open-meteo.com"
	defaultLLMBaseURL = "https://api.anthropic.com"
	defaultModel      = "claude-3-5-sonnet-latest"
)

var httpClient = &http.Client{Timeout: 20 * time.Second}

type input struct {
	TaskID   string `json:"task_id"`
	Prompt   string `json:"prompt"`
	Location string `json:"location"`
	State    string `json:"state"`
	Country  string `json:"country"`
	Days     int    `json:"days"`
}

type output struct {
	TaskID  string        `json:"task_id,omitempty"`
	Status  string        `json:"status"`
	Result  string        `json:"result,omitempty"`
	Report  weatherReport `json:"report,omitempty"`
	Warning string        `json:"warning,omitempty"`
	Error   string        `json:"error,omitempty"`
}

type weatherReport struct {
	Location          string        `json:"location"`
	Days              int           `json:"days"`
	GeneratedAt       string        `json:"generated_at"`
	Forecast          []dayForecast `json:"forecast"`
	RawSummary        string        `json:"raw_summary"`
	LLMSummary        string        `json:"llm_summary"`
	SourceAttribution string        `json:"source_attribution"`
}

type dayForecast struct {
	Date       string  `json:"date"`
	Weather    string  `json:"weather"`
	HighC      float64 `json:"high_c"`
	LowC       float64 `json:"low_c"`
	PrecipMM   float64 `json:"precip_mm"`
	WindMaxKPH float64 `json:"wind_max_kph"`
}

type geocodeResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Admin1    string  `json:"admin1"`
		Country   string  `json:"country"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timezone  string  `json:"timezone"`
	} `json:"results"`
}

type forecastResponse struct {
	Daily struct {
		Time             []string  `json:"time"`
		WeatherCode      []int     `json:"weather_code"`
		Temperature2mMax []float64 `json:"temperature_2m_max"`
		Temperature2mMin []float64 `json:"temperature_2m_min"`
		PrecipitationSum []float64 `json:"precipitation_sum"`
		WindSpeed10mMax  []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string                 `json:"role"`
	Content []anthropicTextContent `json:"content"`
}

type anthropicTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func main() {
	in, err := readInput()
	if err != nil {
		write(output{Status: "failed", Error: err.Error()})
		os.Exit(1)
	}

	location := strings.TrimSpace(in.Location)
	if location == "" {
		location = defaultLocation
	}
	state := strings.TrimSpace(in.State)
	if state == "" {
		state = defaultState
	}
	country := strings.TrimSpace(in.Country)
	if country == "" {
		country = defaultCountry
	}
	days := in.Days
	if days <= 0 || days > 14 {
		days = defaultDays
	}

	resolved, err := geocode(location, state, country)
	if err != nil {
		write(output{
			TaskID: in.TaskID,
			Status: "failed",
			Error:  fmt.Sprintf("geocode lookup failed: %v", err),
		})
		os.Exit(1)
	}

	forecast, err := fetchForecast(resolved.Latitude, resolved.Longitude, resolved.Timezone, days)
	if err != nil {
		write(output{
			TaskID: in.TaskID,
			Status: "failed",
			Error:  fmt.Sprintf("forecast lookup failed: %v", err),
		})
		os.Exit(1)
	}

	rawSummary := composeForecastNarrative(resolved.DisplayName, forecast)
	llmSummary, llmErr := summarizeWithLLM(rawSummary, in.Prompt)
	warning := ""
	if llmErr != nil {
		warning = fmt.Sprintf("llm summary unavailable: %v; using local summary fallback", llmErr)
		llmSummary = fallbackSummary(resolved.DisplayName, forecast)
	}

	write(output{
		TaskID: in.TaskID,
		Status: "completed",
		Result: fmt.Sprintf("generated %d-day weather report for %s", len(forecast), resolved.DisplayName),
		Report: weatherReport{
			Location:          resolved.DisplayName,
			Days:              len(forecast),
			GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
			Forecast:          forecast,
			RawSummary:        rawSummary,
			LLMSummary:        llmSummary,
			SourceAttribution: "Weather data from Open-Meteo (https://open-meteo.com/).",
		},
		Warning: warning,
	})
}

func readInput() (input, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return input{}, fmt.Errorf("read input: %w", err)
	}

	var in input
	if len(strings.TrimSpace(string(data))) == 0 {
		return in, nil
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return input{}, fmt.Errorf("invalid json input: %w", err)
	}
	return in, nil
}

type resolvedLocation struct {
	DisplayName string
	Latitude    float64
	Longitude   float64
	Timezone    string
}

func geocode(city, state, country string) (resolvedLocation, error) {
	base := strings.TrimRight(envOrDefault("WEATHER_GEOCODE_BASE_URL", defaultGeoBaseURL), "/")
	endpoint := base + "/v1/search"

	q := url.Values{}
	q.Set("name", city)
	q.Set("count", "1")
	q.Set("language", "en")
	q.Set("format", "json")
	if country != "" {
		q.Set("country", country)
	}
	if state != "" {
		q.Set("admin1", state)
	}

	var resp geocodeResponse
	if err := fetchJSON(endpoint+"?"+q.Encode(), &resp); err != nil {
		return resolvedLocation{}, err
	}
	if len(resp.Results) == 0 {
		return resolvedLocation{}, fmt.Errorf("no geocode result for %s", city)
	}

	r := resp.Results[0]
	display := strings.TrimSpace(strings.Join([]string{r.Name, r.Admin1, r.Country}, ", "))
	if display == "" {
		display = city
	}
	tz := r.Timezone
	if tz == "" {
		tz = "America/Denver"
	}

	return resolvedLocation{
		DisplayName: display,
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
		Timezone:    tz,
	}, nil
}

func fetchForecast(lat, lon float64, timezone string, days int) ([]dayForecast, error) {
	base := strings.TrimRight(envOrDefault("WEATHER_FORECAST_BASE_URL", defaultWxBaseURL), "/")
	endpoint := base + "/v1/forecast"

	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	q.Set("timezone", timezone)
	q.Set("forecast_days", strconv.Itoa(days))
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,wind_speed_10m_max")

	var resp forecastResponse
	if err := fetchJSON(endpoint+"?"+q.Encode(), &resp); err != nil {
		return nil, err
	}

	count := minLen(
		len(resp.Daily.Time),
		len(resp.Daily.WeatherCode),
		len(resp.Daily.Temperature2mMax),
		len(resp.Daily.Temperature2mMin),
		len(resp.Daily.PrecipitationSum),
		len(resp.Daily.WindSpeed10mMax),
	)
	if count == 0 {
		return nil, fmt.Errorf("forecast response missing daily data")
	}

	out := make([]dayForecast, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, dayForecast{
			Date:       resp.Daily.Time[i],
			Weather:    weatherCodeDescription(resp.Daily.WeatherCode[i]),
			HighC:      round1(resp.Daily.Temperature2mMax[i]),
			LowC:       round1(resp.Daily.Temperature2mMin[i]),
			PrecipMM:   round1(resp.Daily.PrecipitationSum[i]),
			WindMaxKPH: round1(resp.Daily.WindSpeed10mMax[i]),
		})
	}

	return out, nil
}

func composeForecastNarrative(location string, days []dayForecast) string {
	var b strings.Builder
	fmt.Fprintf(&b, "5-day weather report for %s\n", location)
	for _, d := range days {
		fmt.Fprintf(
			&b,
			"- %s: %s, high %.1fC, low %.1fC, precip %.1fmm, max wind %.1fkm/h\n",
			d.Date,
			d.Weather,
			d.HighC,
			d.LowC,
			d.PrecipMM,
			d.WindMaxKPH,
		)
	}
	return strings.TrimSpace(b.String())
}

func summarizeWithLLM(rawSummary, userPrompt string) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}

	model := strings.TrimSpace(envOrDefault("ANTHROPIC_MODEL", defaultModel))
	baseURL := strings.TrimRight(envOrDefault("ANTHROPIC_BASE_URL", defaultLLMBaseURL), "/")
	endpoint := baseURL + "/v1/messages"

	prompt := "Summarize this weather report for a non-technical user. " +
		"Keep it concise, include major risks (rain/wind/temp swings), and a practical 5-day recommendation.\n\n" +
		rawSummary
	if strings.TrimSpace(userPrompt) != "" {
		prompt = prompt + "\n\nAdditional request from user: " + strings.TrimSpace(userPrompt)
	}

	reqBody := anthropicRequest{
		Model:       model,
		MaxTokens:   350,
		Temperature: 0.2,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []anthropicTextContent{
					{Type: "text", Text: prompt},
				},
			},
		},
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("build llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out anthropicResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	for _, c := range out.Content {
		if c.Type == "text" && strings.TrimSpace(c.Text) != "" {
			return strings.TrimSpace(c.Text), nil
		}
	}

	return "", fmt.Errorf("llm response did not include text content")
}

func fallbackSummary(location string, days []dayForecast) string {
	if len(days) == 0 {
		return "No forecast data available."
	}
	high := days[0].HighC
	low := days[0].LowC
	windiest := days[0]
	wetDays := 0

	for _, d := range days {
		if d.HighC > high {
			high = d.HighC
		}
		if d.LowC < low {
			low = d.LowC
		}
		if d.WindMaxKPH > windiest.WindMaxKPH {
			windiest = d
		}
		if d.PrecipMM >= 2 {
			wetDays++
		}
	}

	return fmt.Sprintf(
		"%s outlook: highs around %.1fC, lows around %.1fC over the next %d days. "+
			"Expect %d wetter day(s), with the strongest wind on %s (%.1f km/h). "+
			"Plan outdoor activity around lower-rain windows and carry layers for morning/evening temperature swings.",
		location,
		high,
		low,
		len(days),
		wetDays,
		windiest.Date,
		windiest.WindMaxKPH,
	)
}

func weatherCodeDescription(code int) string {
	switch code {
	case 0:
		return "clear sky"
	case 1, 2:
		return "partly cloudy"
	case 3:
		return "overcast"
	case 45, 48:
		return "fog"
	case 51, 53, 55, 56, 57:
		return "drizzle"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "rain"
	case 71, 73, 75, 77, 85, 86:
		return "snow"
	case 95, 96, 99:
		return "thunderstorm"
	default:
		return fmt.Sprintf("weather code %d", code)
	}
}

func fetchJSON(endpoint string, target any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

func minLen(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func write(out output) {
	_ = json.NewEncoder(os.Stdout).Encode(out)
}
