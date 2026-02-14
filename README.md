# Rime CLI

The official command-line interface for [Rime](https://rime.ai) text-to-speech synthesis. Generate natural-sounding speech from text, stream audio in real-time, and integrate Rime TTS into your workflows.

![rime tts demo](docs/gifs/tts-streaming.gif)

## Install

### Homebrew (macOS & Linux)

```bash
brew tap rimelabs/rime-cli
brew install rime-cli
```

### Shell Script

```bash
curl -fsSL https://rime.ai/install-cli.sh | sh
```

To install a specific version:

```bash
curl -fsSL https://rime.ai/install-cli.sh | sh -s v1.0.0
```

### From Source

```bash
go install github.com/rimelabs/rime-cli@latest
```

## Quick Start

```bash
# Authenticate with your Rime API key
rime login

# Synthesize text to speech — streams and plays immediately
rime tts "Hello from Rime" --speaker astra --model-id arcana

# Save to a file instead
rime tts "Hello from Rime" --speaker astra --model-id arcana -o hello.wav

# Play a quick demo
rime hello
```

## Commands

### `rime login`

Opens the Rime dashboard in your browser and saves your API key locally.

```bash
rime login
```

### `rime tts TEXT`

Synthesize text to speech. Streams audio and plays it in real-time as it arrives.

```bash
rime tts "Your text here" --speaker astra --model-id arcana
```

![Streaming TTS](docs/gifs/tts-streaming.gif)

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--speaker` | `-s` | Voice to use (e.g. `astra`, `celeste`) |
| `--model-id` | `-m` | Model ID (`arcana` for WAV, `mistv2` for MP3) |
| `--output` | `-o` | Save to file (use `-` for stdout) |
| `--play` | `-p` | Play audio after saving to file |
| `--lang` | `-l` | Language code (default: `eng`) |
| `--json` | | Output results as JSON |
| `--quiet` | `-q` | Suppress non-essential output |

![Save and play](docs/gifs/tts-save-play.gif)

### `rime curl [TEXT]`

Generate a curl command for making TTS API requests. Useful for debugging and integration.

```bash
rime curl "Hello world" --speaker astra --model-id arcana --oneline
```

![curl demo](docs/gifs/curl-demo.gif)

### `rime play FILE`

Play a WAV or MP3 audio file with waveform visualization.

```bash
rime play output.wav
```

![Play demo](docs/gifs/play-demo.gif)

### `rime hello`

Quick demo that plays a time-appropriate greeting using the Astra voice.

```bash
rime hello
```

![Hello demo](docs/gifs/hello-demo.gif)

### `rime uninstall`

Print removal instructions for the CLI binary and configuration files.

## Models

| Model | Format | Description |
|-------|--------|-------------|
| `arcana` | WAV | High-quality, low-latency |
| `mistv2` | MP3 | Compressed output |
| `mist` | MP3 | Legacy |

## Languages

**Arcana:** English (`eng`), Arabic (`ara`), French (`fra`), German (`ger`), Hebrew (`heb`), Hindi (`hin`), Japanese (`jpn`), Portuguese (`por`), Sinhala (`sin`), Spanish (`spa`), Tamil (`tam`)

**Mist / MistV2:** English (`eng`), French (`fra`), German (`ger`), Spanish (`spa`)

## Configuration

| Item | Location |
|------|----------|
| API key | `~/.rime/cli-api-token` |
| Config directory | `~/.rime/` |

The `RIME_CLI_API_KEY` environment variable takes precedence over the stored key.

## Uninstall

**Homebrew:**

```bash
brew uninstall rime-cli
brew untap rimelabs/rime-cli
```

**Shell install:**

```bash
rm ~/.rime/bin/rime
```

**Remove configuration:**

```bash
rm -rf ~/.rime
```

## License

Proprietary. See [rime.ai](https://rime.ai) for terms.
