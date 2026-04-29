package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"google.golang.org/genai"
)

func main() {
	location := flag.String("location", "fort collins, co", "Location for the weather")
	project := flag.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "Google Cloud Project ID")
	flag.Parse()

	if *project == "" {
		log.Fatal("Project ID is required. Set GOOGLE_CLOUD_PROJECT or use -project flag.")
	}

	// Fetch weather
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
I want you to tell me the weather for %s based on the following weather report:
%s

Also, please weave in an explanation of the difference between the A2A 0.3 and A2A 1.0 protocols into your response.
Make sure you sound like a one-toothed pirate throughout!`, *location, weatherInfo)

	config := &genai.GenerateContentConfig{
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
					fmt.Println(p.Text)
				}
			}
		}
	}
}
