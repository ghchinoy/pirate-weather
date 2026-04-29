package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- fetchWeather tests ---

func TestFetchWeather_Success(t *testing.T) {
	expected := "Sunny +57F"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := fmt.Fprint(w, expected); err != nil {
			t.Errorf("test server write error: %v", err)
		}
	}))
	defer srv.Close()

	// Temporarily override the base URL to point at our test server.
	original := wttrBaseURL
	wttrBaseURL = srv.URL
	defer func() { wttrBaseURL = original }()

	got, err := fetchWeather("fort collins, co")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestFetchWeather_URLEncoding(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.RawPath
		if capturedPath == "" {
			capturedPath = r.URL.Path
		}
		if _, err := fmt.Fprint(w, "ok"); err != nil {
			t.Errorf("test server write error: %v", err)
		}
	}))
	defer srv.Close()

	original := wttrBaseURL
	wttrBaseURL = srv.URL
	defer func() { wttrBaseURL = original }()

	_, err := fetchWeather("fort collins, co")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spaces should be encoded as + or %20 in the path, not left raw.
	if strings.Contains(capturedPath, " ") {
		t.Errorf("URL path was not encoded, got: %s", capturedPath)
	}
}

func TestFetchWeather_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	original := wttrBaseURL
	wttrBaseURL = srv.URL
	defer func() { wttrBaseURL = original }()

	// A non-200 response still returns the body without error from http.Get;
	// we're verifying it at least doesn't panic and returns a result.
	result, err := fetchWeather("fort collins, co")
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result even for server error response")
	}
}

func TestFetchWeather_ConnectionRefused(t *testing.T) {
	original := wttrBaseURL
	wttrBaseURL = "http://127.0.0.1:1" // nothing listening here
	defer func() { wttrBaseURL = original }()

	_, err := fetchWeather("anywhere")
	if err == nil {
		t.Fatal("expected an error for refused connection, got nil")
	}
}

// --- parseWeatherResponse tests ---

func TestParseWeatherResponse_Valid(t *testing.T) {
	raw := `{
		"ascii_art":       "  \\   /  Sunny",
		"primary_color":   "#FFD700",
		"secondary_color": "#FF8C00",
		"pirate_response": "Arrgh, it be sunny!"
	}`

	wr, err := parseWeatherResponse(raw)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if wr.PrimaryColor != "#FFD700" {
		t.Errorf("PrimaryColor: expected #FFD700, got %q", wr.PrimaryColor)
	}
	if wr.SecondaryColor != "#FF8C00" {
		t.Errorf("SecondaryColor: expected #FF8C00, got %q", wr.SecondaryColor)
	}
	if wr.PirateResponse != "Arrgh, it be sunny!" {
		t.Errorf("PirateResponse: unexpected value %q", wr.PirateResponse)
	}
	if !strings.Contains(wr.ASCIIArt, "Sunny") {
		t.Errorf("ASCIIArt: expected 'Sunny', got %q", wr.ASCIIArt)
	}
}

func TestParseWeatherResponse_InvalidJSON(t *testing.T) {
	_, err := parseWeatherResponse("this is not json {{{")
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestParseWeatherResponse_EmptyJSON(t *testing.T) {
	// An empty object is valid JSON — fields should just be zero values.
	wr, err := parseWeatherResponse("{}")
	if err != nil {
		t.Fatalf("expected no error for empty JSON object, got: %v", err)
	}
	if wr.PrimaryColor != "" || wr.ASCIIArt != "" {
		t.Errorf("expected empty fields for empty JSON, got: %+v", wr)
	}
}

func TestParseWeatherResponse_EmptyString(t *testing.T) {
	_, err := parseWeatherResponse("")
	if err == nil {
		t.Fatal("expected an error for empty string, got nil")
	}
}

// --- buildPrompt tests ---

func TestBuildPrompt_ContainsLocation(t *testing.T) {
	location := "tortuga"
	prompt := buildPrompt(location, "sunny skies")
	if !strings.Contains(prompt, location) {
		t.Errorf("prompt does not contain location %q", location)
	}
}

func TestBuildPrompt_ContainsWeatherInfo(t *testing.T) {
	weatherInfo := "Sunny +72F, calm winds"
	prompt := buildPrompt("anywhere", weatherInfo)
	if !strings.Contains(prompt, weatherInfo) {
		t.Errorf("prompt does not contain weather info %q", weatherInfo)
	}
}

func TestBuildPrompt_ContainsPirateInstructions(t *testing.T) {
	prompt := buildPrompt("anywhere", "cloudy")
	if !strings.Contains(prompt, "pirate") {
		t.Error("prompt does not mention 'pirate'")
	}
	if !strings.Contains(prompt, "A2A") {
		t.Error("prompt does not mention 'A2A'")
	}
	if !strings.Contains(prompt, "ASCII art") {
		t.Error("prompt does not mention 'ASCII art'")
	}
	if !strings.Contains(prompt, "hex color code") {
		t.Error("prompt does not mention 'hex color code'")
	}
}
