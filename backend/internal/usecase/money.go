package usecase

import (
	"fmt"
	"strings"
)

// formatNaira renders an amount for notification/error text the same way the
// frontend's formatMoney() does: "₦" + thousand-grouped digits, with decimals
// shown only when the amount isn't a whole number.
func formatNaira(amount float64) string {
	str := fmt.Sprintf("%.2f", amount)
	intPart, decPart, _ := strings.Cut(str, ".")

	neg := strings.HasPrefix(intPart, "-")
	if neg {
		intPart = intPart[1:]
	}

	var grouped []byte
	for i, c := range []byte(intPart) {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			grouped = append(grouped, ',')
		}
		grouped = append(grouped, c)
	}

	result := string(grouped)
	if decPart != "00" {
		result += "." + decPart
	}
	if neg {
		result = "-" + result
	}
	return "₦" + result
}
