package usecase

// GigPurse's pricing model (Scenario B — dual-sided commission, chosen over
// commission-only and commission+subscription in the pricing model): the
// talent's agreed price is what they'd take home with zero fees; GigPurse
// takes a cut from both sides on release.
//
// The client pays everything into TrustVault in one go. On release it all
// lands in GigPurse's own PayPetal wallet; the talent's share is paid out of
// that and GigPurse keeps the rest:
//
//	agreed price P, talent commission 10%, client service fee 5%
//	  → client pays           = 1.05 * P
//	  → talent take-home      = 0.90 * P
//	  → GigPurse keeps        = 0.15 * P   (0.10P + 0.05P)
const (
	TalentCommissionRate = 0.10
	ClientServiceFeeRate = 0.05
)

// TalentTakeHome is what the talent is paid — the agreed price minus
// GigPurse's commission.
func TalentTakeHome(agreedPrice float64) float64 {
	return agreedPrice * (1 - TalentCommissionRate)
}

// PlatformFee is GigPurse's total revenue on one payment — the talent's
// commission plus the client's service fee.
func PlatformFee(agreedPrice float64) float64 {
	return agreedPrice * (TalentCommissionRate + ClientServiceFeeRate)
}

// ClientTotal is what the client actually pays — the agreed price plus
// their service fee (equivalently: TalentTakeHome + PlatformFee).
func ClientTotal(agreedPrice float64) float64 {
	return agreedPrice * (1 + ClientServiceFeeRate)
}
