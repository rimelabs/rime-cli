package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/rimelabs/rime-cli/internal/api"
)

func TestGenerateCurlCommand_Basic(t *testing.T) {
	opts := CurlOptions{
		Text:    "hello world",
		Speaker: "astra",
		ModelID: "arcana",
		Lang:    "eng",
		APIURL:  "https://api.example.com",
	}

	cmd, err := generateCurlCommand(opts, &api.TTSOptions{})
	if err != nil {
		t.Fatalf("generateCurlCommand failed: %v", err)
	}

	if !strings.Contains(cmd, "curl") {
		t.Error("curl command should contain 'curl'")
	}
	if !strings.Contains(cmd, "hello world") {
		t.Error("curl command should contain text")
	}
	if !strings.Contains(cmd, "astra") {
		t.Error("curl command should contain speaker")
	}
	if !strings.Contains(cmd, "arcana") || !strings.Contains(cmd, "modelId") {
		t.Error("curl command should contain model-id (as modelId in JSON)")
	}
	if !strings.Contains(cmd, "https://api.example.com") {
		t.Error("curl command should contain API URL")
	}
	if !strings.Contains(cmd, "$RIME_CLI_API_KEY") {
		t.Error("curl command should use placeholder when showKey is false")
	}
	if !strings.Contains(cmd, "--output 'output.wav'") {
		t.Error("curl command should contain --output flag")
	}
	if !strings.Contains(cmd, "--fail") {
		t.Error("curl command should contain --fail flag")
	}
}

func TestGenerateCurlCommand_Oneline(t *testing.T) {
	opts := CurlOptions{
		Text:    "test",
		Speaker: "astra",
		ModelID: "arcana",
		Oneline: true,
		APIURL:  "https://api.example.com",
	}

	cmd, err := generateCurlCommand(opts, &api.TTSOptions{})
	if err != nil {
		t.Fatalf("generateCurlCommand failed: %v", err)
	}

	if strings.Contains(cmd, "\\\n") {
		t.Error("oneline mode should not contain line breaks")
	}
	if !strings.Contains(cmd, "curl -X POST") {
		t.Error("oneline mode should use -X POST format")
	}
	if !strings.Contains(cmd, "-o 'output.wav'") {
		t.Error("oneline mode should contain -o flag")
	}
	if !strings.Contains(cmd, " -f ") {
		t.Error("oneline mode should contain -f flag")
	}
}

func TestAudioFormatToExt(t *testing.T) {
	if ext := audioFormatToExt("audio/mp3"); ext != "mp3" {
		t.Errorf("expected mp3, got %s", ext)
	}
	if ext := audioFormatToExt("audio/wav"); ext != "wav" {
		t.Errorf("expected wav, got %s", ext)
	}
	if ext := audioFormatToExt(""); ext != "wav" {
		t.Errorf("expected wav for unknown format, got %s", ext)
	}
}

func TestGenerateCurlCommand_MP3Model(t *testing.T) {
	opts := CurlOptions{
		Text:    "test",
		Speaker: "luna",
		ModelID: "mistv2",
		APIURL:  "https://api.example.com",
	}
	modelOpts := &api.TTSOptions{}

	cmd, err := generateCurlCommand(opts, modelOpts)
	if err != nil {
		t.Fatalf("generateCurlCommand failed: %v", err)
	}

	if !strings.Contains(cmd, "--output 'output.mp3'") {
		t.Error("mp3 model should produce --output output.mp3")
	}
}

func TestCurlCmd_MissingRequiredFlags(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	cmd := NewCurlCmd()
	cmd.SetArgs([]string{"test text"})

	err := cmd.Execute()
	if err == nil {
		t.Error("curl command with text but no speaker/model-id should fail")
	}
}

func TestCurlCmd_InvalidLangWithoutText(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	cmd := NewCurlCmd()
	cmd.SetArgs([]string{"-l", "english", "-s", "astra", "-m", "arcana"})

	err := cmd.Execute()
	if err == nil {
		t.Error("curl command with invalid language should fail even without text")
	}
	if !strings.Contains(err.Error(), "invalid language") {
		t.Errorf("expected error about invalid language, got: %v", err)
	}
}

func TestGenerateCurlCommand_NewParams(t *testing.T) {
	temp := 0.7
	topP := 0.9
	opts := CurlOptions{
		Text:    "hello",
		Speaker: "astra",
		ModelID: "arcana",
		Lang:    "eng",
		APIURL:  "https://api.example.com",
	}
	modelOpts := &api.TTSOptions{
		Temperature: &temp,
		TopP:        &topP,
	}

	curlCmd, err := generateCurlCommand(opts, modelOpts)
	if err != nil {
		t.Fatalf("generateCurlCommand failed: %v", err)
	}

	if !strings.Contains(curlCmd, "temperature") {
		t.Error("curl command should contain temperature field")
	}
	if !strings.Contains(curlCmd, "top_p") {
		t.Error("curl command should contain top_p field")
	}
}

func TestGenerateCurlCommand_MistParams(t *testing.T) {
	pause := true
	speed := "1.0,1.2"
	opts := CurlOptions{
		Text:    "hello",
		Speaker: "astra",
		ModelID: "mistv2",
		Lang:    "eng",
		APIURL:  "https://api.example.com",
	}
	modelOpts := &api.TTSOptions{
		PauseBetweenBrackets: &pause,
		InlineSpeedAlpha:     &speed,
	}

	curlCmd, err := generateCurlCommand(opts, modelOpts)
	if err != nil {
		t.Fatalf("generateCurlCommand failed: %v", err)
	}

	if !strings.Contains(curlCmd, "pauseBetweenBrackets") {
		t.Error("curl command should contain pauseBetweenBrackets field")
	}
	if !strings.Contains(curlCmd, "inlineSpeedAlpha") {
		t.Error("curl command should contain inlineSpeedAlpha field")
	}
}
