package usecase

// GigPurse's pricing model (Scenario B — dual-sided commission, chosen over
// commission-only and commission+subscription in the pricing model): the
// talent's agreed price is what they'd take home with zero fees; GigPurse
// takes a cut from both sides on release.
//
// PayPetal's TrustCore agreements have no partial-release capability — the
// full `amount` on an agreement always goes to the counterparty in one
// shot, with no way for GigPurse to skim a cut from that release. The only
// platform-revenue mechanism PayPetal actually offers is `merchantCharge`:
// an amount added on top of `amount`, billed to the payer, that goes to
// GigPurse separately (confirmed live against sandbox — the checkout page
// shows amount+merchantCharge as one total, and the customer is only ever
// charged that combined total).
//
// So "talent pays a 10% commission" isn't a deduction on release — it's
// baked in by escrowing only 90% of the agreed price in the first place;
// GigPurse's total revenue (10% + 5%) is collected as one merchantCharge on
// top of that 90%, landing on exactly the same client-total and
// talent-take-home the pricing model calls for:
//
//	agreed price P, talent commission 10%, client service fee 5%
//	  → escrowed amount (talent take-home) = 0.90 * P
//	  → merchantCharge (platform revenue)  = 0.15 * P   (0.10P + 0.05P)
//	  → client total                       = 1.05 * P   (amount + merchantCharge)
const (
	TalentCommissionRate = 0.10
	ClientServiceFeeRate = 0.05
)

// TalentTakeHome is what actually gets escrowed and released to the talent
// — the agreed price minus GigPurse's commission.
func TalentTakeHome(agreedPrice float64) float64 {
	return agreedPrice * (1 - TalentCommissionRate)
}

// PlatformFee is GigPurse's total revenue on one payment — the talent's
// commission plus the client's service fee, collected in one merchantCharge.
func PlatformFee(agreedPrice float64) float64 {
	return agreedPrice * (TalentCommissionRate + ClientServiceFeeRate)
}

// ClientTotal is what the client actually pays — the agreed price plus
// their service fee (equivalently: TalentTakeHome + PlatformFee).
func ClientTotal(agreedPrice float64) float64 {
	return agreedPrice * (1 + ClientServiceFeeRate)
}
