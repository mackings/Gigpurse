package memory

import (
	"context"
	"fmt"
	"sync"

	"gigpurse/internal/paypetal"
)

// PayPetalFake is an in-memory stand-in for the real PayPetal client, used
// only by tests — it tracks agreements in a map instead of making real HTTP
// calls. SimulatePaymentCompleted stands in for the initiator actually
// completing PayPetal's hosted checkout, which a unit test can't drive for
// real.
type PayPetalFake struct {
	mu         sync.Mutex
	payouts    map[string]bool // customerID -> has a payout account on file
	agreements map[string]*paypetal.AgreementState
	nextID     int
}

func NewPayPetalFake() *PayPetalFake {
	return &PayPetalFake{
		payouts:    make(map[string]bool),
		agreements: make(map[string]*paypetal.AgreementState),
	}
}

var _ paypetal.API = (*PayPetalFake)(nil)

func (f *PayPetalFake) CreateCustomer(ctx context.Context, fullname, email, phone string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	return fmt.Sprintf("cust_fake_%d", f.nextID), nil
}

func (f *PayPetalFake) UpdateCustomer(ctx context.Context, customerID, fullname, email, phone string) error {
	return nil
}

func (f *PayPetalFake) ListCustomers(ctx context.Context) ([]paypetal.Customer, error) {
	return nil, nil
}

func (f *PayPetalFake) ListBanks(ctx context.Context) ([]paypetal.Bank, error) {
	return []paypetal.Bank{{Name: "Fake Test Bank", Code: "000"}}, nil
}

func (f *PayPetalFake) ValidateBankAccount(ctx context.Context, accountNumber, bankCode string) (string, error) {
	return "Test Account Holder", nil
}

func (f *PayPetalFake) LinkPayoutAccount(ctx context.Context, customerID, accountNumber, bankCode string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payouts[customerID] = true
	return nil
}

func (f *PayPetalFake) CreateTrustCoreAgreement(ctx context.Context, in paypetal.CreateAgreementInput) (*paypetal.AgreementResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.payouts[in.CounterpartyCustomerID] {
		return nil, &paypetal.APIError{HTTPStatus: 400, Code: "missing_payout_account", Message: "the receiver has no payout account on file"}
	}
	f.nextID++
	txID := fmt.Sprintf("txn_fake_%d", f.nextID)
	f.agreements[in.ReferenceID] = &paypetal.AgreementState{
		Reference: in.ReferenceID, TransactionID: txID,
		Status: "PENDING", PayoutStatus: "NONE", RefundStatus: "NONE",
	}
	return &paypetal.AgreementResult{
		PaymentURL: "https://fake.paypetal.test/pay?ref=" + in.ReferenceID, TransactionID: txID,
		Reference: in.ReferenceID, Status: "PENDING",
	}, nil
}

func (f *PayPetalFake) GetTrustCoreAgreement(ctx context.Context, reference string) (*paypetal.AgreementState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.agreements[reference]
	if !ok {
		return nil, &paypetal.APIError{HTTPStatus: 400, Code: "agreement_not_found", Message: "no agreement found for reference " + reference}
	}
	state := *a
	return &state, nil
}

func (f *PayPetalFake) CompleteTrustCoreAgreement(ctx context.Context, reference string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.agreements[reference]
	if !ok {
		return &paypetal.APIError{HTTPStatus: 400, Code: "agreement_not_found", Message: "no agreement found for reference " + reference}
	}
	a.PayoutStatus = "COMPLETED"
	return nil
}

func (f *PayPetalFake) RefundTrustCoreAgreement(ctx context.Context, reference string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.agreements[reference]
	if !ok {
		return &paypetal.APIError{HTTPStatus: 400, Code: "agreement_not_found", Message: "no agreement found for reference " + reference}
	}
	a.RefundStatus = "COMPLETED"
	return nil
}

// SimulatePaymentCompleted flips a pending agreement to ONGOING — standing
// in for the initiator completing PayPetal's hosted checkout.
func (f *PayPetalFake) SimulatePaymentCompleted(reference string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if a, ok := f.agreements[reference]; ok {
		a.Status = "ONGOING"
	}
}
