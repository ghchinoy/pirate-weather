// Package main implements a CLI that fetches weather data and delivers it
// as a one-toothed pirate using Vertex AI Gemini, with lipgloss terminal rendering.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"google.golang.org/genai"
)

// errNoContent is returned when the Gemini model response contains no usable text.
var errNoContent = fmt.Errorf("no text content in model response")

const (
	defaultLocation = "fort collins, co"
	defaultModel    = "gemini-3.1-flash-lite-preview"
	vertexLocation  = "global"
	httpTimeout     = 10 * time.Second
)

// wttrBaseURL is the base URL for the wttr.in weather service.
// Defined as a var so it can be overridden in tests.
var wttrBaseURL = "https://wttr.in"

// WeatherResponse holds the structured JSON output from the Gemini model.
type WeatherResponse struct {
	ASCIIArt       string `json:"ascii_art"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	PirateResponse string `json:"pirate_response"`
}

func main() {
	location := flag.String("location", defaultLocation, "Location for the weather")
	project := flag.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "Google Cloud Project ID")
	flag.Parse()

	if *project == "" {
		log.Fatal("Project ID is required. Set GOOGLE_CLOUD_PROJECT or use -project flag.")
	}

	weatherInfo, err := fetchWeather(*location)
	if err != nil {
		log.Fatalf("Failed to fetch weather: %v", err)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  *project,
		Location: vertexLocation,
	})
	if err != nil {
		log.Fatalf("Failed to create GenAI client: %v", err)
	}

	weatherResp, err := generatePirateWeather(ctx, client, *location, weatherInfo)
	if err != nil {
		log.Fatalf("Failed to generate pirate weather: %v", err)
	}

	renderWeatherBox(*location, weatherResp)
}

// fetchWeather retrieves the plain-text weather report from wttr.in for the given location.
func fetchWeather(location string) (string, error) {
	weatherURL := fmt.Sprintf("%s/%s?0AT", wttrBaseURL, url.QueryEscape(location))

	httpClient := &http.Client{Timeout: httpTimeout}

	resp, err := httpClient.Get(weatherURL) //nolint:noctx // simple CLI, no request context needed
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", weatherURL, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("warning: failed to close response body: %v", cerr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	return string(body), nil
}

// buildPrompt constructs the pirate weather prompt for the given location and weather data.
func buildPrompt(location, weatherInfo string) string {
	return fmt.Sprintf(`You are Antigravity, a pirate with only one tooth.
Process the following weather report for %s:
%s

Tasks:
1. Create a beautiful, multi-line ASCII art representation of this weather condition. Make it look great in a terminal.
2. Select a primary hex color code that represents the weather (e.g., #FFD700 for sunny, #AAAAAA for cloudy).
3. Select a secondary hex color code as an accent.
4. Write your response telling me the weather, and weave in an explanation of the difference between the A2A 0.3 and A2A 1.0 protocols. Sound like a one-toothed pirate throughout!`, location, weatherInfo)
}

// parseWeatherResponse unmarshals a JSON string into a WeatherResponse.
func parseWeatherResponse(raw string) (*WeatherResponse, error) {
	var wr WeatherResponse
	if err := json.Unmarshal([]byte(raw), &wr); err != nil {
		return nil, fmt.Errorf("unmarshal response JSON: %w", err)
	}
	return &wr, nil
}

// generatePirateWeather sends the weather data to Gemini and returns a structured WeatherResponse.
func generatePirateWeather(ctx context.Context, client *genai.Client, location, weatherInfo string) (*WeatherResponse, error) {
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"ascii_art": {
					Type:        genai.TypeString,
					Description: "A creative, multi-line ASCII art representation of the current weather.",
				},
				"primary_color": {
					Type:        genai.TypeString,
					Description: "A hex color code representing the primary color of the weather (e.g. #FFD700 for sun).",
				},
				"secondary_color": {
					Type:        genai.TypeString,
					Description: "A hex color code representing a secondary accent color.",
				},
				"pirate_response": {
					Type:        genai.TypeString,
					Description: "The pirate's spoken response including the weather and the A2A 0.3 vs 1.0 explanation.",
				},
			},
			Required: []string{"ascii_art", "primary_color", "secondary_color", "pirate_response"},
		},
		Tools: []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		},
	}

	result, err := client.Models.GenerateContent(ctx, defaultModel, genai.Text(buildPrompt(location, weatherInfo)), config)
	if err != nil {
		return nil, fmt.Errorf("GenerateContent: %w", err)
	}

	for _, c := range result.Candidates {
		if c.Content == nil {
			continue
		}
		for _, p := range c.Content.Parts {
			if p.Text == "" {
				continue
			}
			return parseWeatherResponse(p.Text)
		}
	}

	return nil, errNoContent
}

// renderWeatherBox prints the AI-generated ASCII art in a lipgloss-styled terminal box,
// followed by the pirate's spoken response.
func renderWeatherBox(location string, w *WeatherResponse) {
	weatherBoxStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(w.PrimaryColor)).
		Background(lipgloss.Color("#1A2B4C")).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(w.SecondaryColor))

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(w.SecondaryColor)).
		Bold(true).
		Underline(true).
		MarginBottom(1)

	header := headerStyle.Render(fmt.Sprintf("🏴‍☠️  Captain's Weather Log: %s 🏴‍☠️", strings.ToUpper(location)))
	box := weatherBoxStyle.Render(fmt.Sprintf("%s\n%s", header, w.ASCIIArt))

	fmt.Println(box)
	fmt.Println()
	fmt.Println(w.PirateResponse)
}
