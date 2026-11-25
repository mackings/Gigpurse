package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gigpurse/internal/paypetal"
)

// PayPetalFake is an in-memory stand-in for the real PayPetal client, used
// only by tests — it tracks agreements in a map instead of making real HTTP
// calls. SimulateVaultPaid stands in for the client actually completing
// PayPetal's hosted checkout, which a unit test can't drive for real.
type PayPetalFake struct {
	mu        sync.Mutex
	customers []paypetal.Customer
	vaults    map[string]*vaultFake
	LastVault paypetal.CreateVaultInput
	VaultDown bool
	// RejectAccount is an account number the bank lookup refuses.
	RejectAccount string
	nextID        int
}

func NewPayPetalFake() *PayPetalFake {
	return &PayPetalFake{}
}

var _ paypetal.API = (*PayPetalFake)(nil)

func (f *PayPetalFake) CreateCustomer(ctx context.Context, fullname, email, phone string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := fmt.Sprintf("cust_fake_%d", f.nextID)
	f.customers = append(f.customers, paypetal.Customer{CustomerID: id, Fullname: fullname, Email: email, PhoneNumber: phone})
	return id, nil
}

func (f *PayPetalFake) CustomerExists(ctx context.Context, customerID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.customers {
		if c.CustomerID == customerID {
			return true, nil
		}
	}
	return false, nil
}

// ForgetCustomers drops every customer — standing in for GigPurse being pointed at a different PayPetal environment, where IDs
// stored from the old one mean nothing.
func (f *PayPetalFake) ForgetCustomers() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.customers = nil
}

func (f *PayPetalFake) UpdateCustomer(ctx context.Context, customerID, fullname, email, phone string) error {
	return nil
}

func (f *PayPetalFake) ListCustomers(ctx context.Context) ([]paypetal.Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]paypetal.Customer(nil), f.customers...), nil
}

func (f *PayPetalFake) ListBanks(ctx context.Context) ([]paypetal.Bank, error) {
	return []paypetal.Bank{{Name: "Fake Test Bank", Code: "000"}}, nil
}

func (f *PayPetalFake) ValidateBankAccount(ctx context.Context, accountNumber, bankCode string) (string, error) {
	if accountNumber == f.RejectAccount {
		return "", &paypetal.APIError{HTTPStatus: 400, Code: "request_failed", Message: "Account verification failed"}
	}
	return "Test Account Holder", nil
}

// VaultCode is the release code the fake "emails" for every agreement.
const VaultCode = "483921"

type vaultFake struct {
	status  string
	coded   bool
	failErr error // returned by the next release...
	applied bool  // ...which nonetheless went through
}

func (f *PayPetalFake) vault(reference string) (*vaultFake, error) {
	v, ok := f.vaults[reference]
	if !ok {
		return nil, &paypetal.APIError{HTTPStatus: 404, Code: "agreement_not_found", Message: "Agreement not found"}
	}
	return v, nil
}

func (f *PayPetalFake) CreateVaultAgreement(ctx context.Context, in paypetal.CreateVaultInput) (*paypetal.VaultResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.vaults == nil {
		f.vaults = map[string]*vaultFake{}
	}
	if _, ok := f.vaults[in.ReferenceID]; !ok {
		f.vaults[in.ReferenceID] = &vaultFake{status: "CREATED"}
	}
	f.LastVault = in
	return &paypetal.VaultResult{PaymentURL: "https://fake.paypetal.test/pay?pid=" + in.ReferenceID, Reference: "pp_" + in.ReferenceID, Status: f.vaults[in.ReferenceID].status}, nil
}

func (f *PayPetalFake) GetVaultAgreement(ctx context.Context, reference string) (*paypetal.VaultState, error) {
	// A real round-trip takes long enough for two concurrent finalizes to
	// both read state before either writes; an instant fake would hide that.
	time.Sleep(5 * time.Millisecond)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.VaultDown {
		return nil, fmt.Errorf("paypetal: request failed: timeout")
	}
	v, err := f.vault(reference)
	if err != nil {
		return nil, err
	}
	return &paypetal.VaultState{Status: v.status}, nil
}

func (f *PayPetalFake) SendVaultReleaseCode(ctx context.Context, reference string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, err := f.vault(reference)
	if err != nil {
		return err
	}
	if v.status != "PAID" {
		return &paypetal.APIError{HTTPStatus: 400, Code: "agreement_has_not_been_paid", Message: "Agreement has not been paid"}
	}
	v.coded = true
	return nil
}

func (f *PayPetalFake) ReleaseVaultAgreement(ctx context.Context, reference, code string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, err := f.vault(reference)
	if err != nil {
		return err
	}
	if v.failErr != nil {
		err := v.failErr
		v.failErr = nil
		if v.applied {
			v.status = "COMPLETED"
		}
		return err
	}
	switch {
	case v.status == "COMPLETED":
		return &paypetal.APIError{HTTPStatus: 400, Code: "agreement_has_already_been_completed", Message: "Agreement has already been completed"}
	case !v.coded:
		return &paypetal.APIError{HTTPStatus: 400, Code: "code_expired", Message: "Code expired"}
	case code != VaultCode:
		return &paypetal.APIError{HTTPStatus: 400, Code: "invalid_code", Message: "Invalid code"}
	}
	v.status = "COMPLETED"
	return nil
}

// SimulateVaultPaid stands in for the client completing hosted checkout.
func (f *PayPetalFake) SimulateVaultPaid(reference string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vaults[reference].status = "PAID"
}

// FailNextVaultRelease makes the next release return an error, with applied
// saying whether PayPetal actually moved the money anyway.
func (f *PayPetalFake) FailNextVaultRelease(reference string, applied bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v := f.vaults[reference]
	v.failErr, v.applied = fmt.Errorf("paypetal: request failed: timeout"), applied
}

// SetVaultDown makes status lookups fail.
func (f *PayPetalFake) SetVaultDown(down bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.VaultDown = down
}
