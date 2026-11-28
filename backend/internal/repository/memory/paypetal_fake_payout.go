package memory

import (
	"context"
	"fmt"
	"strconv"

	"gigpurse/internal/paypetal"
)

// PayoutFee is what the fake charges the sub-wallet on top of each payout.
const PayoutFee = 10.0

type trustCoreFake struct {
	state paypetal.TrustCore
}

// FakeSubWallet is the payout sub-wallet's state, exported so tests can top
// it up and check it.
type FakeSubWallet struct {
	paypetal.SubWallet
	Exists bool
}

func (f *PayPetalFake) init() {
	if f.trustcores == nil {
		f.trustcores = map[string]*trustCoreFake{}
		f.linked = map[string]string{}
	}
}

func (f *PayPetalFake) GetSubWallet(ctx context.Context, reference string) (*paypetal.SubWallet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.SubWallet.Exists || f.SubWallet.RecordID != reference {
		return nil, paypetal.ErrNotFound
	}
	sw := f.SubWallet.SubWallet
	return &sw, nil
}

func (f *PayPetalFake) CreateSubWallet(ctx context.Context, name, reference string) (*paypetal.SubWallet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.SubWallet = FakeSubWallet{Exists: true, SubWallet: paypetal.SubWallet{
		ID: "sw_fake_1", RecordID: reference, Name: name, Currency: "NGN",
		BankName: "9Payment (9PSB)", AccountNumber: "5031681901", AccountName: "GIGPURSE-" + name,
	}}
	sw := f.SubWallet.SubWallet
	return &sw, nil
}

// TopUp credits the payout sub-wallet, as a bank transfer into it would.
func (f *PayPetalFake) TopUp(amount float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.SubWallet.Balance += amount
}

func (f *PayPetalFake) SubWalletBalance() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.SubWallet.Balance
}

func (f *PayPetalFake) LinkPayoutAccount(ctx context.Context, customerID, accountNumber, bankCode string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.init()
	f.linked[customerID] = bankCode + "/" + accountNumber
	return nil
}

// LinkedAccount is the bank account TrustCore would pay customerID.
func (f *PayPetalFake) LinkedAccount(customerID string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.linked[customerID]
}

func (f *PayPetalFake) CreateSubWalletPayout(ctx context.Context, in paypetal.SubWalletPayout) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.init()
	f.Creates++
	if f.PayoutDown {
		return fmt.Errorf("paypetal: request failed: timeout")
	}
	lose := f.LoseCreateResponse
	f.LoseCreateResponse = false
	if _, ok := f.trustcores[in.ReferenceID]; ok {
		return nil // agreement_exists — the real client treats it as success
	}
	if in.SubWalletID != f.SubWallet.ID || in.RedirectURL == "" || in.Description == "" {
		return &paypetal.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "bad payout request"}
	}
	amount, err := strconv.ParseFloat(in.Amount, 64)
	if err != nil || amount <= 0 {
		return &paypetal.APIError{HTTPStatus: 400, Code: "invalid_amount", Message: "invalid amount"}
	}
	if f.SubWallet.Balance < amount+PayoutFee {
		return &paypetal.APIError{HTTPStatus: 400, Code: "insufficient_subwallet_balance", Message: "The subwallet's balance is too low to fund this escrow."}
	}
	f.SubWallet.Balance -= amount + PayoutFee
	f.trustcores[in.ReferenceID] = &trustCoreFake{state: paypetal.TrustCore{
		Reference: in.ReferenceID, Status: "ONGOING", Amount: amount, CustomerID: in.CustomerID,
		PayoutStatus: "NONE", RefundStatus: "NONE",
	}}
	if lose {
		return fmt.Errorf("paypetal: request failed: timeout")
	}
	return nil
}

// SeedTrustCore plants an agreement under reference, as if something else
// had used it.
func (f *PayPetalFake) SeedTrustCore(reference string, amount float64, customerID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.init()
	f.trustcores[reference] = &trustCoreFake{state: paypetal.TrustCore{
		Reference: reference, Status: "ONGOING", Amount: amount, CustomerID: customerID, PayoutStatus: "NONE", RefundStatus: "NONE",
	}}
}

// TrustCoreCount is how many agreements exist.
func (f *PayPetalFake) TrustCoreCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.trustcores)
}

func (f *PayPetalFake) trustcore(reference string) (*trustCoreFake, error) {
	f.init()
	t, ok := f.trustcores[reference]
	if !ok {
		return nil, &paypetal.APIError{HTTPStatus: 404, Code: "agreement_not_found", Message: "No agreement found"}
	}
	return t, nil
}

func (f *PayPetalFake) GetTrustCore(ctx context.Context, reference string) (*paypetal.TrustCore, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PayoutDown {
		return nil, fmt.Errorf("paypetal: request failed: timeout")
	}
	t, err := f.trustcore(reference)
	if err != nil {
		return nil, paypetal.ErrNotFound
	}
	s := t.state
	return &s, nil
}

func (f *PayPetalFake) CompleteTrustCore(ctx context.Context, reference string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, err := f.trustcore(reference)
	if err != nil {
		return err
	}
	switch t.state.PayoutStatus {
	case "PENDING", "PROCESSING":
		return &paypetal.APIError{HTTPStatus: 400, Code: "payout_in_progress", Message: "A payout for this agreement is already in progress."}
	case "COMPLETED":
		return &paypetal.APIError{HTTPStatus: 400, Code: "payout_already_completed", Message: "The payout for this agreement has already been completed."}
	}
	if t.state.RefundStatus != "NONE" {
		return &paypetal.APIError{HTTPStatus: 400, Code: "already_refunded", Message: "This agreement has been refunded and cannot be paid out."}
	}
	if f.linked[t.state.CustomerID] == "" {
		return &paypetal.APIError{HTTPStatus: 400, Code: "missing_payout_account", Message: "The receiver does not have a payout account."}
	}
	f.Completes++
	t.state.PayoutStatus = "PENDING"
	return nil
}

func (f *PayPetalFake) RefundTrustCore(ctx context.Context, reference string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, err := f.trustcore(reference)
	if err != nil {
		return err
	}
	switch {
	case t.state.PayoutStatus == "COMPLETED":
		return &paypetal.APIError{HTTPStatus: 400, Code: "payout_completed", Message: "This agreement has already been paid out and cannot be refunded."}
	case t.state.PayoutStatus == "PENDING" || t.state.PayoutStatus == "PROCESSING":
		return &paypetal.APIError{HTTPStatus: 400, Code: "payout_in_progress", Message: "A payout is in progress."}
	case t.state.RefundStatus != "NONE":
		return &paypetal.APIError{HTTPStatus: 400, Code: "already_refunded", Message: "This agreement has already been refunded."}
	}
	t.state.RefundStatus = "COMPLETED"
	f.SubWallet.Balance += t.state.Amount // PayPetal's fee isn't refunded
	return nil
}

// SettleTransfer finishes the bank transfer in flight for reference, as
// COMPLETED, FAILED or REFUNDED.
func (f *PayPetalFake) SettleTransfer(reference, outcome string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := f.trustcores[reference]
	t.state.PayoutStatus = outcome
	if outcome == "COMPLETED" {
		t.state.Status = "COMPLETED"
	}
}

// TrustCoreState is what PayPetal holds for reference, or nil.
func (f *PayPetalFake) TrustCoreState(reference string) *paypetal.TrustCore {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.init()
	if t, ok := f.trustcores[reference]; ok {
		s := t.state
		return &s
	}
	return nil
}
