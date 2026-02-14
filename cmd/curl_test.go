package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateCurlCommand_Basic(t *testing.T) {
	opts := CurlOptions{
		Text:    "hello world",
		Speaker: "astra",
		ModelID: "arcana",
		Lang:    "eng",
		APIURL:  "https://api.example.com",
	}

	cmd, err := generateCurlCommand(opts)
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
}

func TestGenerateCurlCommand_Oneline(t *testing.T) {
	opts := CurlOptions{
		Text:    "test",
		Speaker: "astra",
		ModelID: "arcana",
		Oneline: true,
		APIURL:  "https://api.example.com",
	}

	cmd, err := generateCurlCommand(opts)
	if err != nil {
		t.Fatalf("generateCurlCommand failed: %v", err)
	}

	if strings.Contains(cmd, "\\\n") {
		t.Error("oneline mode should not contain line breaks")
	}
	if !strings.Contains(cmd, "curl -X POST") {
		t.Error("oneline mode should use -X POST format")
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
