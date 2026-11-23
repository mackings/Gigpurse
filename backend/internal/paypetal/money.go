package paypetal

import "strconv"

// FormatAmount renders a Naira amount the way PayPetal expects it: a string
// in whole Naira with two decimal places ("500.00" is ₦500), not kobo.
func FormatAmount(naira float64) string {
	return strconv.FormatFloat(naira, 'f', 2, 64)
}
