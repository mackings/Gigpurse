package usecase

import "testing"

func TestFilterContent_PhoneNumbers(t *testing.T) {
	masked := []string{
		"08031234567", "call 0803 123 4567 now", "0803-123-4567", "0803.123.4567", "0 8 0 3 1 2 3 4 5 6 7",
		"+2348031234567", "+234 803 123 4567", "234 803 123 4567", "+234 (0) 803 123 4567", "2348031234567",
		"my number:08031234567!", "Call me.08031234567", "+2348031234567 2026", "07012345678", "09112345678",
		"+44 7911 123456", "+1 (415) 555-2671",
	}
	for _, in := range masked {
		out := filterContent(in)
		digits := 0
		for i := 0; i < len(out); i++ {
			if isDigit(out[i]) {
				digits++
			}
		}
		if out == in || digits > 4 {
			t.Errorf("%q was not masked: %q", in, out)
		}
	}

	kept := []string{
		"₦150,000 for the night", "500000 naira", "1,080,912,345,678", "price 12500.50", "on 2026-11-23 at 10:00",
		"09-11-2026 1000 guests", "09/11/2026 10:30", "from 10:00 to 18:30", "invoice 1234567890123", "we need 12 people, 3 songs",
		"+10% on top", "ref 6720a1b2c3d4e5f607080910",
	}
	for _, in := range kept {
		if out := filterContent(in); out != in {
			t.Errorf("%q should be untouched, got %q", in, out)
		}
	}
}
