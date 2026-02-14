//go:build headless

package playback

import "fmt"

func RunNonInteractivePlay(filepath string) error {
	return fmt.Errorf("audio playback not available in headless build")
}

func PlayAudioData(data []byte, contentType string) error {
	return fmt.Errorf("audio playback not available in headless build")
}

func IsPlaybackEnabled() bool {
	return false
}
