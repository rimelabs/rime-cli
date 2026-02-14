package visualizer

// Braille waveform visualization using two rows of characters.
//
// Top row: bars grow UPWARD from bottom edge
// Bottom row: mirror of top, bars grow DOWNWARD from top edge
//
// Together they create a vertically symmetric waveform.
//
// Braille cell dot positions and bit values:
//
//	┌───┬───┐     Bits:
//	│ 1 │ 4 │     1, 8
//	├───┼───┤
//	│ 2 │ 5 │     2, 16
//	├───┼───┤
//	│ 3 │ 6 │     4, 32
//	├───┼───┤
//	│ 7 │ 8 │     64, 128
//	└───┴───┘
//
// Codepoint: U+2800 + bitmask

const brailleBase = 0x2800

// Bit values for each dot
const (
	dot1 = 1
	dot2 = 2
	dot3 = 4
	dot4 = 8
	dot5 = 16
	dot6 = 32
	dot7 = 64
	dot8 = 128
)

// topLeftBits: level 0-4, bars grow up from bottom (dot 7 → 3 → 2 → 1)
var topLeftBits = []int{
	0,                         // level 0
	dot7,                      // level 1
	dot7 | dot3,               // level 2
	dot7 | dot3 | dot2,        // level 3
	dot7 | dot3 | dot2 | dot1, // level 4
}

// topRightBits: level 0-4, bars grow up from bottom (dot 8 → 6 → 5 → 4)
var topRightBits = []int{
	0,                         // level 0
	dot8,                      // level 1
	dot8 | dot6,               // level 2
	dot8 | dot6 | dot5,        // level 3
	dot8 | dot6 | dot5 | dot4, // level 4
}

// botLeftBits: level 0-4, bars grow down from top (dot 1 → 2 → 3 → 7)
var botLeftBits = []int{
	0,                         // level 0
	dot1,                      // level 1
	dot1 | dot2,               // level 2
	dot1 | dot2 | dot3,        // level 3
	dot1 | dot2 | dot3 | dot7, // level 4
}

// botRightBits: level 0-4, bars grow down from top (dot 4 → 5 → 6 → 8)
var botRightBits = []int{
	0,                         // level 0
	dot4,                      // level 1
	dot4 | dot5,               // level 2
	dot4 | dot5 | dot6,        // level 3
	dot4 | dot5 | dot6 | dot8, // level 4
}

// TopChar returns a braille char for the top row (bars grow upward).
func TopChar(leftLevel, rightLevel int) rune {
	bits := topLeftBits[clamp(leftLevel)] | topRightBits[clamp(rightLevel)]
	return rune(brailleBase + bits)
}

// BotChar returns a braille char for the bottom row (bars grow downward).
func BotChar(leftLevel, rightLevel int) rune {
	bits := botLeftBits[clamp(leftLevel)] | botRightBits[clamp(rightLevel)]
	return rune(brailleBase + bits)
}

// QuantizeAmplitude converts amplitude (0.0-1.0) to level (0-4).
func QuantizeAmplitude(amp float64) int {
	if amp <= 0 {
		return 0
	}
	if amp >= 1 {
		return 4
	}
	return int(amp * 5)
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 4 {
		return 4
	}
	return v
}
