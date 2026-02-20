package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"time"
)

const (
	defaultAPIBaseURL   = "https://users.rime.ai/v1/rime-tts"
	defaultDashboardURL = "https://app.rime.ai"

	ModelIDArcana   = "arcana"
	ModelIDArcanaV2 = "arcanav2"
	ModelIDMistV2   = "mistv2"
	ModelIDMist     = "mist"
)

var validModelIDs = map[string]bool{
	ModelIDArcana:   true,
	ModelIDArcanaV2: true,
	ModelIDMistV2:   true,
	ModelIDMist:     true,
}

type langEntry struct {
	iso3 string
	iso1 string
}

var allLangs = []langEntry{
	{"ara", "ar"},
	{"eng", "en"},
	{"fra", "fr"},
	{"ger", "de"},
	{"heb", "he"},
	{"hin", "hi"},
	{"jpn", "ja"},
	{"por", "pt"},
	{"sin", "si"},
	{"spa", "es"},
	{"tam", "ta"},
}

var mistLangSet = map[string]bool{
	"eng": true, "en": true,
	"fra": true, "fr": true,
	"ger": true, "de": true,
	"spa": true, "es": true,
}

var arcanaV2LangSet = map[string]bool{
	"eng": true, "en": true,
	"spa": true, "es": true,
	"ger": true, "de": true,
	"fra": true, "fr": true,
	"hin": true, "hi": true,
}

var arcanaLangSet map[string]bool

func init() {
	arcanaLangSet = make(map[string]bool, len(allLangs)*2)
	for _, l := range allLangs {
		arcanaLangSet[l.iso3] = true
		arcanaLangSet[l.iso1] = true
	}
}

func langSetForModel(modelID string) map[string]bool {
	if modelID == ModelIDMist || modelID == ModelIDMistV2 {
		return mistLangSet
	}
	if modelID == ModelIDArcanaV2 {
		return arcanaV2LangSet
	}
	return arcanaLangSet
}

func IsValidLang(lang string, modelID string) bool {
	return langSetForModel(modelID)[lang]
}

func ValidLangsForModel(modelID string) []string {
	set := langSetForModel(modelID)
	langs := make([]string, 0, len(set))
	for k := range set {
		langs = append(langs, k)
	}
	sort.Strings(langs)
	return langs
}

type Client struct {
	baseURL   string
	apiKey    string
	userAgent string
	client    *http.Client
}

type TTSRequest struct {
	Text    string `json:"text"`
	Speaker string `json:"speaker"`
	ModelID string `json:"modelId,omitempty"`
	Lang    string `json:"lang,omitempty"`
}

type TTSOptions struct {
	Speaker     string
	ModelID     string
	Lang        string
	AudioFormat string
}

func IsValidModelID(modelID string) bool {
	return validModelIDs[modelID]
}

func GetAudioFormat(modelID string) string {
	if modelID == ModelIDMistV2 || modelID == ModelIDMist {
		return "audio/mp3"
	}
	return "audio/wav"
}

func GetAPIURL() string {
	if url := os.Getenv("RIME_API_URL"); url != "" {
		return url
	}
	return defaultAPIBaseURL
}

func GetDashboardURL() string {
	if url := os.Getenv("RIME_DASHBOARD_URL"); url != "" {
		return url
	}
	return defaultDashboardURL
}

func NewClient(apiKey string, version string, baseURL ...string) *Client {
	userAgent := fmt.Sprintf("rime-cli/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
	url := GetAPIURL()
	if len(baseURL) > 0 && baseURL[0] != "" {
		url = baseURL[0]
	}
	return &Client{
		baseURL:   url,
		apiKey:    apiKey,
		userAgent: userAgent,
		client:    &http.Client{},
	}
}

func (c *Client) TTS(text string, opts *TTSOptions) ([]byte, error) {
	if opts == nil || opts.Speaker == "" {
		return nil, fmt.Errorf("speaker is required")
	}
	if opts.ModelID == "" {
		return nil, fmt.Errorf("modelId is required")
	}
	if !IsValidModelID(opts.ModelID) {
		return nil, fmt.Errorf("invalid modelId: %s (valid options: %s, %s, %s, %s)", opts.ModelID, ModelIDArcana, ModelIDArcanaV2, ModelIDMistV2, ModelIDMist)
	}

	reqBody := TTSRequest{
		Text:    text,
		Speaker: opts.Speaker,
		ModelID: opts.ModelID,
		Lang:    opts.Lang,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	audioFormat := opts.AudioFormat
	if audioFormat == "" {
		audioFormat = GetAudioFormat(opts.ModelID)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", audioFormat)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("authentication failed: invalid API key")
		case http.StatusBadRequest:
			return nil, fmt.Errorf("invalid request: %s", string(body))
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("rate limited: too many requests")
		default:
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return audio, nil
}

type TTSStreamResult struct {
	Body        io.ReadCloser
	ContentType string
	TTFB        time.Duration
}

func (c *Client) TTSStream(text string, opts *TTSOptions) (*TTSStreamResult, error) {
	if opts == nil || opts.Speaker == "" {
		return nil, fmt.Errorf("speaker is required")
	}
	if opts.ModelID == "" {
		return nil, fmt.Errorf("modelId is required")
	}
	if !IsValidModelID(opts.ModelID) {
		return nil, fmt.Errorf("invalid modelId: %s (modelId should be one of: %s, %s, %s, %s)", opts.ModelID, ModelIDArcana, ModelIDArcanaV2, ModelIDMistV2, ModelIDMist)
	}

	reqBody := TTSRequest{
		Text:    text,
		Speaker: opts.Speaker,
		ModelID: opts.ModelID,
		Lang:    opts.Lang,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	audioFormat := opts.AudioFormat
	if audioFormat == "" {
		audioFormat = GetAudioFormat(opts.ModelID)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", audioFormat)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("User-Agent", c.userAgent)

	start := time.Now()
	resp, err := c.client.Do(req)
	ttfb := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("authentication failed: invalid API key")
		case http.StatusBadRequest:
			return nil, fmt.Errorf("invalid request: %s", string(body))
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("rate limited: too many requests")
		default:
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}
	}

	// To detect empty responses to streaming TTS requests, we can't just check the
	// ContentLength, because it is unknown (-1) so we need to peek at the first byte.
	peekBuf := make([]byte, 1)
	n, err := resp.Body.Read(peekBuf)

	// If the first byte is EOF, the response is empty and we return an error.
	if err == io.EOF && n == 0 {
		resp.Body.Close()
		// formatted to say that speaker {speaker} and language {language} are valid for modelId {modelId}
		return nil, fmt.Errorf("invalid request: server returned empty response. Please double-check that speaker '%s' and language '%s' are valid for modelId '%s'", opts.Speaker, opts.Lang, opts.ModelID)
	}

	contentType := resp.Header.Get("Content-Type")
	return &TTSStreamResult{
		// Since we've consumed a byte, we reconstruct the stream using MultiReader
		// so downstream code can read the full response including the peeked byte.
		Body:        io.NopCloser(io.MultiReader(bytes.NewReader(peekBuf[:n]), resp.Body)),
		ContentType: contentType,
		TTFB:        ttfb,
	}, nil
}

func EffectiveOpts(opts *TTSOptions) (speaker, modelId, lang string) {
	speaker = opts.Speaker
	modelId = opts.ModelID
	lang = opts.Lang
	if lang == "" {
		lang = "eng"
	}
	return
}
