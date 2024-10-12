package util

import "testing"

func TestRandHex(t *testing.T) {
	length := 16
	hexString := RandHex(length)
	//Check the length of the string to match the length of the hex chars (16)
	if len(hexString) != length {
		t.Errorf("RandHex(%d) length = %d, want %d", length, len(hexString), length)
	}

	//Check if there are valid hex chars
	for _, char := range hexString {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("RandHex(%d) Invalid char = %c", length, char)
		}
	}
}

func BenchmarkRandHex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandHex(16)
	}
}
