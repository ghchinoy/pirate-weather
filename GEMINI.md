# GEMINI.md — pirate-weather

This file captures the architectural patterns, SDK conventions, and project-specific rules for the `pirate-weather` CLI. Coding agents working on this project should read this file in full before making changes.

---

## Project Overview

`pirate-weather` is a Go CLI that:
1. Fetches real-time weather data from `wttr.in`
2. Sends that data to Google Vertex AI (Gemini) to generate a structured JSON response containing AI-generated ASCII art, a hex color palette, and a pirate-voice narrative
3. Renders the output in the terminal using `lipgloss` with the AI-chosen colors

---

## Go Conventions

- **Build:** `go build`
- **Test:** `go test ./... -v`
- **Lint:** `golangci-lint run ./...`
- **Dependencies:** `go mod tidy`
- **Run:** `go run main.go -project=$(gcloud config get project)`

The binary is called `pirate-weather`. Do not commit it — it is excluded in `.gitignore`.

---

## Vertex AI / Gemini SDK

### Package
Always use `google.golang.org/genai`, imported as:
```go
import "google.golang.org/genai"
```
Do **not** use `cloud.google.com/go/vertexai/genai`.

### Client setup
```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    Backend:  genai.BackendVertexAI,
    Project:  projectID,     // from -project flag or GOOGLE_CLOUD_PROJECT env var
    Location: "global",      // this project uses "global" region
})
```

### Model
The current model is `gemini-3.1-flash-lite-preview`. It is stored as the constant `defaultModel` in `main.go`. Update that constant if switching models — do not hardcode the model name elsewhere.

### Structured output
This project uses Gemini's structured JSON output via `ResponseSchema`. The field is called `ResponseMIMEType` (not `ResponseMimeType`):

```go
config := &genai.GenerateContentConfig{
    ResponseMIMEType: "application/json",   // NOTE: MIME not Mime
    ResponseSchema: &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "field_name": {
                Type:        genai.TypeString,
                Description: "...",
            },
        },
        Required: []string{"field_name"},
    },
}
```

The corresponding Go struct uses standard `json:"..."` tags. The struct is `WeatherResponse` in `main.go`.

### Google Search tool
The Google Search tool is enabled to allow Gemini to ground responses:
```go
Tools: []*genai.Tool{
    {GoogleSearch: &genai.GoogleSearch{}},
},
```

**Important:** When using both `ResponseSchema` and `GoogleSearch` together, the model may occasionally return empty candidates (no text, no tool calls). The `generatePirateWeather` function handles this by returning `errNoContent`. Retry logic may be needed for production use.

---

## Project Structure

```
pirate-weather/
├── main.go              # All application logic (see function breakdown below)
├── main_test.go         # Unit tests (no integration tests — see Testing section)
├── .golangci.yml        # Linter config (golangci-lint v2 format)
├── .goreleaser.yaml     # GoReleaser config (version: 1 — required for the GH Action)
├── .gitignore           # Excludes binary, vendor, .DS_Store, etc.
├── scripts/
│   └── install.sh       # curl-based install script for downloading the latest release
└── .github/
    └── workflows/
        ├── lint.yml     # CI: runs on push/PR — build, test, lint
        └── release.yml  # CI: runs on git tags — builds and publishes binaries via GoReleaser
```

### Function breakdown in `main.go`

| Function | Purpose |
|---|---|
| `main()` | Parses flags, orchestrates the three steps |
| `fetchWeather(location)` | GETs weather text from `wttr.in` |
| `buildPrompt(location, weatherInfo)` | Constructs the Gemini prompt string |
| `parseWeatherResponse(raw)` | Unmarshals raw JSON string into `WeatherResponse` |
| `generatePirateWeather(ctx, client, location, weatherInfo)` | Calls Gemini, returns `*WeatherResponse` |
| `renderWeatherBox(location, w)` | Renders lipgloss terminal UI and prints the response |

Keep these functions separate. Each one has its own unit tests — do not merge them back into `main()`.

---

## Weather Data Source

Weather is fetched from `wttr.in` using the `?0AT` query format (plain text, single line):
```
https://wttr.in/{url-encoded-location}?0AT
```

The base URL is stored in the package-level **var** `wttrBaseURL` (not a const) so that tests can override it with an `httptest.Server`. Do not change it to a const.

---

## Lipgloss Rendering

Colors come directly from the Gemini model response (`PrimaryColor` and `SecondaryColor` hex fields). The dark navy background `#1A2B4C` is hardcoded as the box background — this is intentional for contrast regardless of weather condition.

---

## Testing

### Philosophy
- **Unit test** everything that can be tested without a live API: `fetchWeather`, `buildPrompt`, `parseWeatherResponse`
- **Do not unit test** `generatePirateWeather` directly — it requires a live Gemini client. Integration tests for this should use the `//go:build integration` tag
- **Do not unit test** `renderWeatherBox` — it only prints to stdout; cosmetic changes are verified by running the binary

### Running tests
```bash
go test ./... -v
```

### HTTP mocking
Use `net/http/httptest` to mock `wttr.in`:
```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
    if _, err := fmt.Fprint(w, "Sunny +57F"); err != nil {
        t.Errorf("test server write error: %v", err)
    }
}))
defer srv.Close()
wttrBaseURL = srv.URL  // override the package-level var
```

Always restore `wttrBaseURL` with a `defer` after overriding it.

---

## Releasing

Releases are triggered by pushing a git tag:
```bash
git tag v1.x.0
git push origin main
git push origin v1.x.0
```

GoReleaser builds binaries for `linux`, `darwin`, and `windows` on `amd64` and `arm64`.

**Important:** The `.goreleaser.yaml` uses `version: 1`. The GitHub Action pins to the goreleaser v1 release. Do not upgrade to `version: 2` without also upgrading the action to a version that supports it.

---

## Linting

Config is at `.golangci.yml` using golangci-lint **v2** format (requires `version: "2"` at the top).

Key rules enabled: `errcheck`, `govet`, `staticcheck`, `unused`, `ineffassign`, `bodyclose`, `gosec`, `revive`, `gocritic`, `err113`, `contextcheck`.

Suppressions:
- `gosec G107` (URL from variable) is suppressed for `httpClient.Get` — intentional in this CLI
- `err113` dynamic errors are suppressed for `fmt.Errorf` with `%w` wrapping

Always run `golangci-lint run ./...` and ensure **0 issues** before committing.

---

## Authentication

The app uses Application Default Credentials (ADC) automatically via the genai SDK. To authenticate:
```bash
gcloud auth application-default login
```

The Google Cloud project is passed via:
- `-project` flag, **or**
- `GOOGLE_CLOUD_PROJECT` environment variable

Recommended setup:
```bash
export GOOGLE_CLOUD_PROJECT=$(gcloud config get project)
```

Do **not** hardcode project IDs anywhere in the source.
