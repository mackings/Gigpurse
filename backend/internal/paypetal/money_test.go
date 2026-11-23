package paypetal

import "testing"

func TestFormatAmountSendsNairaNotKobo(t *testing.T) {
	cases := map[float64]string{
		500:          "500.00",
		450:          "450.00",
		500 * 0.15:   "75.00",
		19.99:        "19.99",
		450000:       "450000.00",
		333.33 * 0.9: "300.00",
	}
	for in, want := range cases {
		if got := FormatAmount(in); got != want {
			t.Errorf("FormatAmount(%v) = %q, want %q", in, got, want)
		}
	}
}
