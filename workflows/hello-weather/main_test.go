package main

import (
	"strings"
	"testing"
)

func TestComposeForecastNarrative(t *testing.T) {
	days := []dayForecast{
		{Date: "2026-03-01", Weather: "clear sky", HighC: 12.3, LowC: -1.2, PrecipMM: 0.0, WindMaxKPH: 14.2},
		{Date: "2026-03-02", Weather: "rain", HighC: 9.8, LowC: 0.1, PrecipMM: 5.2, WindMaxKPH: 26.0},
	}

	out := composeForecastNarrative("Montrose, Colorado, United States", days)
	if !strings.Contains(out, "Montrose, Colorado, United States") {
		t.Fatalf("missing location in narrative: %q", out)
	}
	if !strings.Contains(out, "2026-03-01") || !strings.Contains(out, "2026-03-02") {
		t.Fatalf("missing day entries in narrative: %q", out)
	}
}

func TestFallbackSummary(t *testing.T) {
	days := []dayForecast{
		{Date: "2026-03-01", HighC: 5.0, LowC: -4.0, PrecipMM: 0.0, WindMaxKPH: 12.0},
		{Date: "2026-03-02", HighC: 10.0, LowC: -2.0, PrecipMM: 3.0, WindMaxKPH: 24.0},
		{Date: "2026-03-03", HighC: 7.0, LowC: -6.0, PrecipMM: 1.5, WindMaxKPH: 8.0},
	}

	out := fallbackSummary("Montrose, Colorado, United States", days)
	if !strings.Contains(out, "Montrose, Colorado, United States outlook") {
		t.Fatalf("missing location summary: %q", out)
	}
	if !strings.Contains(out, "strongest wind on 2026-03-02") {
		t.Fatalf("missing windiest-day detail: %q", out)
	}
}

func TestWeatherCodeDescription(t *testing.T) {
	if weatherCodeDescription(0) != "clear sky" {
		t.Fatalf("unexpected weather code mapping for 0")
	}
	if !strings.Contains(weatherCodeDescription(123), "123") {
		t.Fatalf("expected unknown code text to include code")
	}
}
