package visualizer

import "testing"

func TestTopChar_Empty(t *testing.T) {
	ch := TopChar(0, 0)
	if ch != '⠀' {
		t.Errorf("expected empty braille, got %c (U+%04X)", ch, ch)
	}
}

func TestTopChar_Full(t *testing.T) {
	ch := TopChar(4, 4)
	if ch != '⣿' {
		t.Errorf("expected ⣿, got %c (U+%04X)", ch, ch)
	}
}

func TestBotChar_Full(t *testing.T) {
	ch := BotChar(4, 4)
	if ch != '⣿' {
		t.Errorf("expected ⣿, got %c (U+%04X)", ch, ch)
	}
}

func TestTopChar_Level1(t *testing.T) {
	ch := TopChar(1, 1)
	expected := rune(0x2800 + 64 + 128)
	if ch != expected {
		t.Errorf("expected %c (U+%04X), got %c (U+%04X)", expected, expected, ch, ch)
	}
}

func TestBotChar_Level1(t *testing.T) {
	ch := BotChar(1, 1)
	expected := rune(0x2800 + 1 + 8)
	if ch != expected {
		t.Errorf("expected %c (U+%04X), got %c (U+%04X)", expected, expected, ch, ch)
	}
}

func TestTopChar_AllLevels(t *testing.T) {
	for left := 0; left <= 4; left++ {
		for right := 0; right <= 4; right++ {
			ch := TopChar(left, right)
			if ch < 0x2800 || ch > 0x28FF {
				t.Errorf("TopChar(%d, %d) = %c (U+%04X), not a valid braille character", left, right, ch, ch)
			}
		}
	}
}

func TestBotChar_AllLevels(t *testing.T) {
	for left := 0; left <= 4; left++ {
		for right := 0; right <= 4; right++ {
			ch := BotChar(left, right)
			if ch < 0x2800 || ch > 0x28FF {
				t.Errorf("BotChar(%d, %d) = %c (U+%04X), not a valid braille character", left, right, ch, ch)
			}
		}
	}
}

func TestTopChar_OutOfBounds(t *testing.T) {
	chNeg := TopChar(-1, -1)
	chHigh := TopChar(10, 10)

	chZero := TopChar(0, 0)
	if chNeg != chZero {
		t.Errorf("TopChar(-1, -1) should clamp to 0, got %c", chNeg)
	}
	if chHigh != TopChar(4, 4) {
		t.Errorf("TopChar(10, 10) should clamp to 4, got %c", chHigh)
	}
}

func TestBotChar_OutOfBounds(t *testing.T) {
	chNeg := BotChar(-1, -1)
	chHigh := BotChar(10, 10)

	chZero := BotChar(0, 0)
	if chNeg != chZero {
		t.Errorf("BotChar(-1, -1) should clamp to 0, got %c", chNeg)
	}
	if chHigh != BotChar(4, 4) {
		t.Errorf("BotChar(10, 10) should clamp to 4, got %c", chHigh)
	}
}

func TestQuantizeAmplitude_Zero(t *testing.T) {
	level := QuantizeAmplitude(0.0)
	if level != 0 {
		t.Errorf("QuantizeAmplitude(0.0) = %d, expected 0", level)
	}
}

func TestQuantizeAmplitude_One(t *testing.T) {
	level := QuantizeAmplitude(1.0)
	if level != 4 {
		t.Errorf("QuantizeAmplitude(1.0) = %d, expected 4", level)
	}
}

func TestQuantizeAmplitude_Half(t *testing.T) {
	level := QuantizeAmplitude(0.5)
	if level != 2 {
		t.Errorf("QuantizeAmplitude(0.5) = %d, expected 2", level)
	}
}

func TestQuantizeAmplitude_Negative(t *testing.T) {
	level := QuantizeAmplitude(-0.5)
	if level != 0 {
		t.Errorf("QuantizeAmplitude(-0.5) = %d, expected 0", level)
	}
}

func TestQuantizeAmplitude_AboveOne(t *testing.T) {
	level := QuantizeAmplitude(1.5)
	if level != 4 {
		t.Errorf("QuantizeAmplitude(1.5) = %d, expected 4", level)
	}
}

func TestQuantizeAmplitude_Levels(t *testing.T) {
	tests := []struct {
		amp   float64
		level int
	}{
		{0.0, 0},
		{0.1, 0},
		{0.2, 1},
		{0.4, 2},
		{0.6, 3},
		{0.8, 4},
		{1.0, 4},
	}

	for _, tt := range tests {
		level := QuantizeAmplitude(tt.amp)
		if level != tt.level {
			t.Errorf("QuantizeAmplitude(%f) = %d, expected %d", tt.amp, level, tt.level)
		}
	}
}

func TestSingleLineChar_Level0(t *testing.T) {
	ch := SingleLineChar(0)
	if ch != ' ' {
		t.Errorf("expected space for level 0, got %c", ch)
	}
}

func TestSingleLineChar_Level1(t *testing.T) {
	ch := SingleLineChar(1)
	if ch != '-' {
		t.Errorf("expected '-' for level 1, got %c", ch)
	}
}

func TestSingleLineChar_Level2(t *testing.T) {
	ch := SingleLineChar(2)
	if ch != '⠶' {
		t.Errorf("expected '⠶' for level 2, got %c", ch)
	}
}

func TestSingleLineChar_Level3(t *testing.T) {
	ch := SingleLineChar(3)
	if ch != '⣿' {
		t.Errorf("expected '⣿' for level 3, got %c", ch)
	}
}

func TestSingleLineChar_Level4(t *testing.T) {
	ch := SingleLineChar(4)
	if ch != '⣿' {
		t.Errorf("expected '⣿' for level 4, got %c", ch)
	}
}

func TestSingleLineChar_OutOfBounds(t *testing.T) {
	chNeg := SingleLineChar(-1)
	chHigh := SingleLineChar(10)

	chZero := SingleLineChar(0)
	chMax := SingleLineChar(4)

	if chNeg != chZero {
		t.Errorf("SingleLineChar(-1) should clamp to 0, got %c", chNeg)
	}
	if chHigh != chMax {
		t.Errorf("SingleLineChar(10) should clamp to 4, got %c", chHigh)
	}
}
