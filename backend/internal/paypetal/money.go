package paypetal

import (
	"fmt"
	"math"
	"strconv"
)

// NairaToKobo is the only place in the codebase that knows PayPetal wants
// amounts as a string in the smallest currency unit (kobo for NGN).
// Rounding to the nearest kobo avoids float binary-representation drift
// (e.g. 19.99*100 landing on 1998.9999999).
func NairaToKobo(naira float64) string {
	kobo := int64(math.Round(naira * 100))
	return strconv.FormatInt(kobo, 10)
}

// KoboToNaira is the inverse — used only for logging/reconciliation when
// reading PayPetal's own reported amount back; GigPurse never trusts it
// over its own locally stored amount for a money decision.
func KoboToNaira(kobo string) (float64, error) {
	n, err := strconv.ParseInt(kobo, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid kobo amount %q: %w", kobo, err)
	}
	return float64(n) / 100, nil
}
