# Pirate Weather

A simple Go program that fetches the current weather for a specified location (default: Fort Collins, CO) and uses Google's Vertex AI (Gemini 3.1 Flash Lite Preview) to generate a response as a one-toothed pirate. It also weaves in an explanation of the difference between the A2A 0.3 and A2A 1.0 protocols.

<img width="612" height="435" alt="Image" src="https://github.com/user-attachments/assets/a76bf212-28b0-44ee-a871-081f8550a155" />

## Requirements

- Go 1.21+ (or compatible)
- A Google Cloud Project with the Vertex AI API enabled
- Authenticated with Google Cloud (e.g., via `gcloud auth application-default login`)

## Installation

You can download the latest compiled binary directly using our install script:

```bash
curl -sL https://raw.githubusercontent.com/ghchinoy/pirate-weather/main/scripts/install.sh | bash
```

## Usage

Run the program via the downloaded binary or with `go run main.go`. You need to specify your Google Cloud Project ID, which you can pass via the `-project` flag or the `GOOGLE_CLOUD_PROJECT` environment variable.

```bash
# Set your project ID
export GOOGLE_CLOUD_PROJECT=$(gcloud config get project)

# Run with the default location (Fort Collins, CO)
go run main.go

# Run with a custom location
go run main.go -location "Tortuga"
```

## How it works

1. It makes a request to `wttr.in` to get the plain-text weather data for the specified location.
2. It uses `google.golang.org/genai` to send a prompt along with the weather data to Vertex AI.
3. The LLM processes the data and outputs a colorful, pirate-themed explanation.
