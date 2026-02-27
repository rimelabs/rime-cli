package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

const (
	defaultAPIBaseURL   = "https://users.rime.ai/v1/rime-tts"
	defaultDashboardURL = "https://app.rime.ai"
	oovURL              = "https://beta.rime.ai/oov"

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
	baseURL          string
	apiKey           string
	authHeaderPrefix string
	userAgent        string
	client           *http.Client
}

type TTSRequest struct {
	Text    string `json:"text"`
	Speaker string `json:"speaker"`
	ModelID string `json:"modelId,omitempty"`
	Lang    string `json:"lang,omitempty"`

	// Arcana/ArcanaV2 specific
	RepetitionPenalty *float64 `json:"repetition_penalty,omitempty"`
	Temperature       *float64 `json:"temperature,omitempty"`
	TopP              *float64 `json:"top_p,omitempty"`
	MaxTokens         *int     `json:"max_tokens,omitempty"`

	// Both model families
	SamplingRate *int     `json:"samplingRate,omitempty"`
	SpeedAlpha   *float64 `json:"speedAlpha,omitempty"`

	// Mist/MistV2 specific
	PauseBetweenBrackets     *bool   `json:"pauseBetweenBrackets,omitempty"`
	PhonemizeBetweenBrackets *bool   `json:"phonemizeBetweenBrackets,omitempty"`
	InlineSpeedAlpha         *string `json:"inlineSpeedAlpha,omitempty"`
	NoTextNormalization      *bool   `json:"noTextNormalization,omitempty"`
	SaveOovs                 *bool   `json:"saveOovs,omitempty"`
}

type TTSOptions struct {
	Speaker     string
	ModelID     string
	Lang        string
	AudioFormat string

	// Arcana/ArcanaV2 specific
	RepetitionPenalty *float64
	Temperature       *float64
	TopP              *float64
	MaxTokens         *int

	// Both model families
	SamplingRate *int
	SpeedAlpha   *float64

	// Mist/MistV2 specific
	PauseBetweenBrackets     *bool
	PhonemizeBetweenBrackets *bool
	InlineSpeedAlpha         *string
	NoTextNormalization      *bool
	SaveOovs                 *bool
}

func IsValidModelID(modelID string) bool {
	return validModelIDs[modelID]
}

func IsArcanaModel(modelID string) bool {
	return modelID == ModelIDArcana || modelID == ModelIDArcanaV2
}

func IsMistModel(modelID string) bool {
	return modelID == ModelIDMist || modelID == ModelIDMistV2
}

var validArcanaRates = map[int]bool{
	8000: true, 16000: true, 22050: true, 24000: true,
	44100: true, 48000: true, 96000: true,
}

func ValidateModelParams(opts *TTSOptions) error {
	if opts == nil {
		return nil
	}
	modelID := opts.ModelID

	// Arcana-only params
	if opts.Temperature != nil {
		if !IsArcanaModel(modelID) {
			return fmt.Errorf("--temperature is only supported for arcana/arcanav2 models")
		}
		if *opts.Temperature < 0 || *opts.Temperature > 1 {
			return fmt.Errorf("--temperature must be between 0 and 1, got %g", *opts.Temperature)
		}
	}
	if opts.TopP != nil {
		if !IsArcanaModel(modelID) {
			return fmt.Errorf("--top-p is only supported for arcana/arcanav2 models")
		}
		if *opts.TopP < 0 || *opts.TopP > 1 {
			return fmt.Errorf("--top-p must be between 0 and 1, got %g", *opts.TopP)
		}
	}
	if opts.RepetitionPenalty != nil {
		if !IsArcanaModel(modelID) {
			return fmt.Errorf("--repetition-penalty is only supported for arcana/arcanav2 models")
		}
		if *opts.RepetitionPenalty < 1 || *opts.RepetitionPenalty > 2 {
			return fmt.Errorf("--repetition-penalty must be between 1 and 2, got %g", *opts.RepetitionPenalty)
		}
	}
	if opts.MaxTokens != nil {
		if !IsArcanaModel(modelID) {
			return fmt.Errorf("--max-tokens is only supported for arcana/arcanav2 models")
		}
		if *opts.MaxTokens < 200 || *opts.MaxTokens > 5000 {
			return fmt.Errorf("--max-tokens must be between 200 and 5000, got %d", *opts.MaxTokens)
		}
	}

	// Mist-only params
	if opts.PauseBetweenBrackets != nil && !IsMistModel(modelID) {
		return fmt.Errorf("--pause-between-brackets is only supported for mist/mistv2 models")
	}
	if opts.PhonemizeBetweenBrackets != nil && !IsMistModel(modelID) {
		return fmt.Errorf("--phonemize-between-brackets is only supported for mist/mistv2 models")
	}
	if opts.InlineSpeedAlpha != nil && !IsMistModel(modelID) {
		return fmt.Errorf("--inline-speed-alpha is only supported for mist/mistv2 models")
	}
	if opts.NoTextNormalization != nil && !IsMistModel(modelID) {
		return fmt.Errorf("--no-text-normalization is only supported for mist/mistv2 models")
	}
	if opts.SaveOovs != nil && !IsMistModel(modelID) {
		return fmt.Errorf("--save-oovs is only supported for mist/mistv2 models")
	}

	// Shared params with model-specific validation
	if opts.SpeedAlpha != nil && *opts.SpeedAlpha <= 0 {
		return fmt.Errorf("--speed-alpha must be greater than 0, got %g", *opts.SpeedAlpha)
	}
	if opts.SamplingRate != nil {
		rate := *opts.SamplingRate
		if IsArcanaModel(modelID) {
			if !validArcanaRates[rate] {
				return fmt.Errorf("--sampling-rate for arcana/arcanav2 must be one of 8000, 16000, 22050, 24000, 44100, 48000, 96000; got %d", rate)
			}
		} else if IsMistModel(modelID) {
			if rate < 4000 || rate > 44100 {
				return fmt.Errorf("--sampling-rate for mist/mistv2 must be between 4000 and 44100, got %d", rate)
			}
		}
	}

	return nil
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

type ClientOptions struct {
	APIKey           string
	APIURL           string
	AuthHeaderPrefix string
	Version          string
}

func NewClient(opts ClientOptions) *Client {
	userAgent := UserAgent(opts.Version)
	url := opts.APIURL
	if url == "" {
		url = GetAPIURL()
	}
	authPrefix := opts.AuthHeaderPrefix
	if authPrefix == "" {
		if prefix := os.Getenv("RIME_AUTH_HEADER_PREFIX"); prefix != "" {
			authPrefix = prefix
		} else {
			authPrefix = "Bearer"
		}
	}
	return &Client{
		baseURL:          url,
		apiKey:           opts.APIKey,
		authHeaderPrefix: authPrefix,
		userAgent:        userAgent,
		client:           &http.Client{},
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
	if err := ValidateModelParams(opts); err != nil {
		return nil, err
	}

	reqBody := TTSRequest{
		Text:    text,
		Speaker: opts.Speaker,
		ModelID: opts.ModelID,
		Lang:    opts.Lang,

		RepetitionPenalty:        opts.RepetitionPenalty,
		Temperature:              opts.Temperature,
		TopP:                     opts.TopP,
		MaxTokens:                opts.MaxTokens,
		SamplingRate:             opts.SamplingRate,
		SpeedAlpha:               opts.SpeedAlpha,
		PauseBetweenBrackets:     opts.PauseBetweenBrackets,
		PhonemizeBetweenBrackets: opts.PhonemizeBetweenBrackets,
		InlineSpeedAlpha:         opts.InlineSpeedAlpha,
		NoTextNormalization:      opts.NoTextNormalization,
		SaveOovs:                 opts.SaveOovs,
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
	if c.apiKey != "" && c.authHeaderPrefix != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.authHeaderPrefix, c.apiKey))
	}
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
	if err := ValidateModelParams(opts); err != nil {
		return nil, err
	}

	reqBody := TTSRequest{
		Text:    text,
		Speaker: opts.Speaker,
		ModelID: opts.ModelID,
		Lang:    opts.Lang,

		RepetitionPenalty:        opts.RepetitionPenalty,
		Temperature:              opts.Temperature,
		TopP:                     opts.TopP,
		MaxTokens:                opts.MaxTokens,
		SamplingRate:             opts.SamplingRate,
		SpeedAlpha:               opts.SpeedAlpha,
		PauseBetweenBrackets:     opts.PauseBetweenBrackets,
		PhonemizeBetweenBrackets: opts.PhonemizeBetweenBrackets,
		InlineSpeedAlpha:         opts.InlineSpeedAlpha,
		NoTextNormalization:      opts.NoTextNormalization,
		SaveOovs:                 opts.SaveOovs,
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
	if c.apiKey != "" && c.authHeaderPrefix != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.authHeaderPrefix, c.apiKey))
	}
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

// ValidateAPIKey confirms the API key is valid using the lightweight OOV
// (out-of-vocabulary) endpoint. No TTS credits are consumed.
func (c *Client) ValidateAPIKey() error {
	body := bytes.NewBufferString(`{"text":"cli"}`)
	req, err := http.NewRequest("POST", oovURL, body)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" && c.authHeaderPrefix != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.authHeaderPrefix, c.apiKey))
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication failed: invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
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
