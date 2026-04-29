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

	"github.com/charmbracelet/lipgloss"
	"google.golang.org/genai"
)

type WeatherResponse struct {
	AsciiArt       string `json:"ascii_art"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	PirateResponse string `json:"pirate_response"`
}

func main() {
	location := flag.String("location", "fort collins, co", "Location for the weather")
	project := flag.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "Google Cloud Project ID")
	flag.Parse()

	if *project == "" {
		log.Fatal("Project ID is required. Set GOOGLE_CLOUD_PROJECT or use -project flag.")
	}

	// Fetch weather data as source material
	weatherURL := fmt.Sprintf("https://wttr.in/%s?0AT", url.QueryEscape(*location))
	resp, err := http.Get(weatherURL)
	if err != nil {
		log.Fatalf("Failed to get weather: %v", err)
	}
	defer resp.Body.Close()

	weatherBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read weather data: %v", err)
	}
	weatherInfo := string(weatherBytes)

	// Setup GenAI Client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  *project,
		Location: "global",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	model := "gemini-3.1-flash-lite-preview"
	
	prompt := fmt.Sprintf(`You are Antigravity, a pirate with only one tooth. 
Process the following weather report for %s:
%s

Tasks:
1. Create a beautiful, multi-line ASCII art representation of this weather condition. Make it look great in a terminal.
2. Select a primary hex color code that represents the weather (e.g., #FFD700 for sunny, #AAAAAA for cloudy).
3. Select a secondary hex color code as an accent.
4. Write your response telling me the weather, and weave in an explanation of the difference between the A2A 0.3 and A2A 1.0 protocols. Sound like a one-toothed pirate throughout!`, *location, weatherInfo)

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

	result, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), config)
	if err != nil {
		log.Fatalf("GenerateContent error: %v", err)
	}

	for _, c := range result.Candidates {
		if c.Content != nil {
			for _, p := range c.Content.Parts {
				if p.Text != "" {
					var weatherResp WeatherResponse
					if err := json.Unmarshal([]byte(p.Text), &weatherResp); err != nil {
						log.Fatalf("Failed to unmarshal JSON: %v\nRaw output: %s", err, p.Text)
					}

					// Render with Lipgloss using the AI-chosen colors
					weatherBoxStyle := lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color(weatherResp.PrimaryColor)).
						Background(lipgloss.Color("#1A2B4C")).
						Padding(1, 2).
						MarginTop(1).
						MarginBottom(1).
						Border(lipgloss.RoundedBorder()).
						BorderForeground(lipgloss.Color(weatherResp.SecondaryColor))

					headerStyle := lipgloss.NewStyle().
						Foreground(lipgloss.Color(weatherResp.SecondaryColor)).
						Bold(true).
						Underline(true).
						MarginBottom(1)

					header := headerStyle.Render(fmt.Sprintf("🏴‍☠️  Captain's Weather Log: %s 🏴‍☠️", strings.ToUpper(*location)))
					asciiWeather := weatherBoxStyle.Render(fmt.Sprintf("%s\n%s", header, weatherResp.AsciiArt))

					fmt.Println(asciiWeather)
					fmt.Println()
					fmt.Println(weatherResp.PirateResponse)
				}
			}
		}
	}
}
