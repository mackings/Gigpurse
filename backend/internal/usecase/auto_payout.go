package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"gigpurse/internal/domain"
	"gigpurse/internal/paypetal"
)

// Automatic payouts, as PayPetal's guides lay them out: GigPurse keeps a
// payout sub-wallet topped up, and pays each due payout with a TrustCore
// agreement that sub-wallet funds on the spot, released to the payee's bank.
//
// Never paying twice rests on three things:
//   - each payout has one fixed PayPetal reference (AutoRef), and PayPetal
//     refuses a second agreement under it (agreement_exists) and a second
//     transfer while one is in flight (payout_in_progress);
//   - every step starts by asking PayPetal what state the agreement is in,
//     and a payout only goes back to "due" when PayPetal says no money is
//     set aside for it (no agreement, or one refunded to the sub-wallet);
//   - an admin can't mark a payout paid while PayPetal is handling it.
//
// Nobody has to press anything for a payout to go out: every failure is
// retried on its own, and a payout waiting on a top-up or a new bank account
// goes as soon as that happens. The only stop is a PayPetal record that
// doesn't match the payout (see mismatch), since releasing that could pay the
// wrong person.

const (
	payoutWalletName = "GigPurse Payouts"
	// fundsRecheck is the longest a payout waits on a top-up without trying
	// again anyway, in case reading the wallet's balance fails.
	fundsRecheck    = 30 * time.Minute
	autoStepTimeout = 90 * time.Second
	// balanceFresh is how long a read of the wallet's balance is reused
	// within a pass.
	balanceFresh = 20 * time.Second
)

// retryGaps is the wait after each failure in a row; the last one repeats
// for as long as it takes.
var retryGaps = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour, 3 * time.Hour}

func retryGap(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	if failures > len(retryGaps) {
		failures = len(retryGaps)
	}
	return retryGaps[failures-1]
}

// autoRef is the payout's PayPetal TrustCore reference: "pt" (talent) or
// "pr" (refund) plus the milestone's payment reference. Letters and digits
// only, which PayPetal requires.
func autoRef(p *domain.Payout) string {
	prefix := "pt"
	if p.Kind == domain.PayoutRefund {
		prefix = "pr"
	}
	var b strings.Builder
	for _, r := range p.ReferenceID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	ref := prefix + b.String()
	if p.AutoRound > 0 {
		ref += fmt.Sprintf("n%d", p.AutoRound)
	}
	return ref
}

func (u *payoutUsecase) autoEnabled() bool { return u.subWalletRef != "" }

func ready(at *time.Time) bool { return at == nil || !time.Now().Before(*at) }

func inFlight(status string) bool { return status == "PENDING" || status == "PROCESSING" }

func (u *payoutUsecase) ProcessAuto(ctx context.Context) {
	if !u.autoEnabled() || !u.pass.TryLock() {
		return
	}
	defer u.pass.Unlock()
	u.walletMu.Lock()
	u.balanceAt = time.Time{} // each pass reads the wallet's balance afresh, once
	u.walletMu.Unlock()
	// In-flight ones first, so a pass finishes what it started.
	for _, status := range []string{"sending", "stopping", "due"} {
		rows, err := u.payoutRepo.ListByStatus(ctx, status)
		if err != nil {
			log.Printf("auto payout: listing %s: %v", status, err)
			continue
		}
		for _, p := range rows {
			stepCtx, cancel := context.WithTimeout(ctx, autoStepTimeout)
			u.step(stepCtx, p)
			cancel()
		}
	}
}

// step moves one payout on as far as it can go right now. Callers hold u.pass.
func (u *payoutUsecase) step(ctx context.Context, p *domain.Payout) {
	switch p.Status {
	case "due":
		if p.ManualOnly || !u.canAct(ctx, p) {
			return
		}
		p.Status, p.AutoError, p.RetryAt, p.WaitingOn = "sending", "", nil, ""
		if p.AutoRef == "" {
			p.AutoRef = autoRef(p)
		}
		// Only one claimant wins; an admin marking it paid by hand is the
		// other one.
		if won, err := u.payoutRepo.UpdateIfStatus(ctx, p, "due"); err != nil || !won {
			return
		}
		log.Printf("auto payout %s: sending %s as %s", p.ID, formatNaira(p.Amount), p.AutoRef)
		u.advance(ctx, p, true)
	case "sending", "stopping":
		u.advance(ctx, p, true)
	}
}

// advance acts on what PayPetal says about the payout's agreement.
func (u *payoutUsecase) advance(ctx context.Context, p *domain.Payout, again bool) {
	tc, err := u.client.GetTrustCore(ctx, p.AutoRef)
	if errors.Is(err, paypetal.ErrNotFound) {
		if p.Status == "stopping" {
			u.toManual(ctx, p, "") // nothing was ever set aside
			return
		}
		u.fund(ctx, p, again)
		return
	}
	if err != nil {
		log.Printf("auto payout %s: checking %s: %v", p.ID, p.AutoRef, err)
		return
	}
	if problem := u.mismatch(ctx, p, tc); problem != "" {
		u.hold(ctx, p, problem)
		return
	}
	switch {
	case tc.PayoutStatus == "COMPLETED":
		if u.markPaid(ctx, p, p.Status, "PayPetal "+p.AutoRef, "paypetal") {
			log.Printf("auto payout %s: paid", p.ID)
		}
	case tc.RefundStatus == "COMPLETED":
		msg := ""
		if p.Status == "sending" {
			msg = "PayPetal returned this payout to the sub-wallet. Pay it by hand, or retry it."
		}
		u.toManual(ctx, p, msg)
	case inFlight(tc.PayoutStatus) || inFlight(tc.RefundStatus):
		// The bank transfer, or the refund, is on its way.
	case p.Status == "stopping":
		u.refund(ctx, p)
	case !u.canAct(ctx, p):
		// Waiting for its next try.
	case tc.PayoutStatus == "FAILED" || tc.PayoutStatus == "REFUNDED":
		if p.RetryAt == nil && p.WaitingOn == "" {
			// A new failure: the last send went through to PayPetal, which
			// clears both. Wait before sending again.
			u.failed(ctx, p, "transfer", fmt.Sprintf("The bank transfer didn't go through (%d so far). GigPurse is trying again automatically.", p.AutoAttempts+1))
			return
		}
		log.Printf("auto payout %s: transfer %s, sending again", p.ID, strings.ToLower(tc.PayoutStatus))
		u.send(ctx, p, tc.CustomerID)
	default: // NONE: funded, not yet sent
		u.send(ctx, p, tc.CustomerID)
	}
}

// canAct reports whether a payout may be tried now: it isn't held, and its
// retry time has come, or what it was waiting on has changed.
func (u *payoutUsecase) canAct(ctx context.Context, p *domain.Payout) bool {
	if p.Held {
		return false
	}
	if ready(p.RetryAt) {
		return true
	}
	switch p.WaitingOn {
	case "funds":
		bal, ok := u.walletBalance(ctx)
		return ok && bal > p.BalanceSeen+settlementEpsilon
	case "account":
		user, err := u.userRepo.GetByID(ctx, p.UserID)
		return err == nil && user.PayoutAccount != nil && !user.PayoutAccount.LinkedAt.Equal(p.AccountSeen)
	}
	return false
}

// mismatch guards against a reference collision: an agreement that isn't
// for this payout's amount or payee is never released.
func (u *payoutUsecase) mismatch(ctx context.Context, p *domain.Payout, tc *paypetal.TrustCore) string {
	if math.Abs(tc.Amount-p.Amount) > settlementEpsilon {
		return fmt.Sprintf("PayPetal's agreement %s is for %s, not %s. Check it in the PayPetal dashboard.", p.AutoRef, formatNaira(tc.Amount), formatNaira(p.Amount))
	}
	if user, err := u.userRepo.GetByID(ctx, p.UserID); err == nil && tc.CustomerID != "" && tc.CustomerID != user.PayPetalCustomerID {
		return fmt.Sprintf("PayPetal's agreement %s pays a different customer. Check it in the PayPetal dashboard.", p.AutoRef)
	}
	return ""
}

// fund sets the money aside: a TrustCore agreement funded from the payout
// sub-wallet. Until it exists nothing has moved, so any refusal on the way
// puts the payout back to due.
func (u *payoutUsecase) fund(ctx context.Context, p *domain.Payout, again bool) {
	if !u.canAct(ctx, p) {
		return
	}
	user, err := u.userRepo.GetByID(ctx, p.UserID)
	if err != nil {
		return
	}
	customerID, err := u.ensureCustomer(ctx, user)
	if err != nil {
		u.backToDue(ctx, p, "", "Couldn't set the payee up on PayPetal: "+err.Error()+". GigPurse is trying again automatically.")
		return
	}
	if waitOn, err := u.linkAccount(ctx, p, user, customerID); err != nil {
		u.backToDue(ctx, p, waitOn, err.Error())
		return
	}
	wallet, err := u.payoutWallet(ctx)
	if err != nil {
		u.backToDue(ctx, p, "", "Couldn't reach the payout wallet: "+apiMessage(err)+". GigPurse is trying again automatically.")
		return
	}
	description := "GigPurse payout: " + p.Title
	if p.Kind == domain.PayoutRefund {
		description = "GigPurse refund: " + p.Title
	}
	err = u.client.CreateSubWalletPayout(ctx, paypetal.SubWalletPayout{
		ReferenceID: p.AutoRef, SubWalletID: wallet.ID, CustomerID: customerID,
		Amount: paypetal.FormatAmount(p.Amount), Description: description,
		RedirectURL: u.redirectURL, PayoutRecord: p.ID,
	})
	switch {
	case err == nil:
		if again {
			u.advance(ctx, p, false)
		}
	case paypetal.IsCode(err, "insufficient_subwallet_balance"):
		u.waitForFunds(ctx, p)
	case isRejection(err):
		if paypetal.IsCode(err, "customer_not_found") {
			u.forgetWallet() // the initiator wasn't found: look the sub-wallet up again
		}
		u.backToDue(ctx, p, "", "PayPetal refused the payout: "+apiMessage(err)+". GigPurse is trying again automatically.")
	default:
		// It may have gone through; the next pass asks PayPetal.
		log.Printf("auto payout %s: funding outcome unknown: %v", p.ID, err)
		u.note(ctx, p, "Waiting for PayPetal to confirm the payout.")
	}
}

// send starts (or restarts) the bank transfer, to the account the payee has
// on GigPurse right now.
func (u *payoutUsecase) send(ctx context.Context, p *domain.Payout, customerID string) {
	user, err := u.userRepo.GetByID(ctx, p.UserID)
	if err != nil {
		return
	}
	if waitOn, err := u.linkAccount(ctx, p, user, customerID); err != nil {
		u.failed(ctx, p, waitOn, err.Error())
		return
	}
	err = u.client.CompleteTrustCore(ctx, p.AutoRef)
	switch {
	case err == nil, paypetal.IsCode(err, "payout_in_progress"), paypetal.IsCode(err, "payout_already_completed"):
		p.AutoError, p.RetryAt, p.WaitingOn = "", nil, ""
		_, _ = u.payoutRepo.UpdateIfStatus(ctx, p, p.Status)
	case isRejection(err):
		u.failed(ctx, p, "", "PayPetal didn't start the transfer: "+apiMessage(err)+". GigPurse is trying again automatically.")
	default:
		log.Printf("auto payout %s: transfer outcome unknown: %v", p.ID, err)
	}
}

// linkAccount points PayPetal at the payee's bank account, resolving it
// first as PayPetal requires. On failure it says what to wait on: "account"
// when only the payee can fix it.
func (u *payoutUsecase) linkAccount(ctx context.Context, p *domain.Payout, user *domain.User, customerID string) (string, error) {
	acct := user.PayoutAccount
	if acct == nil {
		u.askForAccount(ctx, p, "you haven't added a bank account")
		return "account", fmt.Errorf("%s has no bank account on GigPurse. It's sent as soon as they add one", user.Name)
	}
	if _, err := u.client.ValidateBankAccount(ctx, acct.AccountNumber, acct.BankCode); err != nil {
		if !isRejection(err) {
			return "", fmt.Errorf("couldn't check %s's bank account (%s). GigPurse is trying again automatically", user.Name, apiMessage(err))
		}
		u.askForAccount(ctx, p, "your bank account couldn't be verified")
		return "account", fmt.Errorf("%s's bank account couldn't be verified. It's sent as soon as they update it, and GigPurse keeps trying meanwhile", user.Name)
	}
	if err := u.client.LinkPayoutAccount(ctx, customerID, acct.AccountNumber, acct.BankCode); err != nil {
		return "", fmt.Errorf("couldn't link %s's bank account on PayPetal (%s). GigPurse is trying again automatically", user.Name, apiMessage(err))
	}
	return "", nil
}

// askForAccount tells the payee, once per wait, that only they can unblock it.
func (u *payoutUsecase) askForAccount(ctx context.Context, p *domain.Payout, why string) {
	if p.WaitingOn == "account" {
		return
	}
	u.tell(ctx, p, "Update your bank account",
		fmt.Sprintf("We couldn't send %s for '%s' because %s. Update it on GigPurse and it goes out automatically.", formatNaira(p.Amount), p.Title, why))
}

func (u *payoutUsecase) refund(ctx context.Context, p *domain.Payout) {
	err := u.client.RefundTrustCore(ctx, p.AutoRef)
	if err != nil && !paypetal.IsCode(err, "already_refunded") && !paypetal.IsCode(err, "refund_in_progress") {
		log.Printf("auto payout %s: refund to sub-wallet: %v", p.ID, err)
		u.note(ctx, p, "PayPetal couldn't return the money to the sub-wallet yet: "+apiMessage(err))
	}
}

// backToDue: only when PayPetal holds no money for the payout. It's tried
// again on its own after the next retry gap.
func (u *payoutUsecase) backToDue(ctx context.Context, p *domain.Payout, waitOn, msg string) {
	from := p.Status
	p.Status = "due"
	u.schedule(ctx, p, waitOn, msg)
	if won, _ := u.payoutRepo.UpdateIfStatus(ctx, p, from); won {
		log.Printf("auto payout %s: back to due, tries again in %s: %s", p.ID, retryGap(p.AutoAttempts), msg)
	}
}

// waitForFunds: the payout wallet is short. Nothing was taken; it goes out
// as soon as the wallet's balance goes up.
func (u *payoutUsecase) waitForFunds(ctx context.Context, p *domain.Payout) {
	from := p.Status
	u.walletMu.Lock()
	u.balanceAt = time.Time{} // the balance just changed, or was misread: read it afresh
	u.walletMu.Unlock()
	bal, ok := u.walletBalance(ctx)
	if !ok {
		bal = math.Inf(1) // can't tell when it's topped up; fall back to fundsRecheck
	}
	at := time.Now().Add(fundsRecheck)
	p.Status, p.WaitingOn, p.BalanceSeen, p.RetryAt = "due", "funds", bal, &at
	p.AutoError = fmt.Sprintf("The payout wallet can't cover %s plus PayPetal's fee. It's sent automatically as soon as the wallet is topped up.", formatNaira(p.Amount))
	if won, _ := u.payoutRepo.UpdateIfStatus(ctx, p, from); won {
		log.Printf("auto payout %s: waiting for the payout wallet to be topped up", p.ID)
	}
}

// failed records a failure where the payout stands and schedules its next try.
func (u *payoutUsecase) failed(ctx context.Context, p *domain.Payout, waitOn, msg string) {
	u.schedule(ctx, p, waitOn, msg)
	_, _ = u.payoutRepo.UpdateIfStatus(ctx, p, p.Status)
}

func (u *payoutUsecase) schedule(ctx context.Context, p *domain.Payout, waitOn, msg string) {
	p.AutoAttempts++
	at := time.Now().Add(retryGap(p.AutoAttempts))
	p.RetryAt, p.WaitingOn, p.AutoError = &at, waitOn, msg
	p.AccountSeen = time.Time{}
	if waitOn == "account" {
		if user, err := u.userRepo.GetByID(ctx, p.UserID); err == nil && user.PayoutAccount != nil {
			p.AccountSeen = user.PayoutAccount.LinkedAt
		}
	}
}

// toManual: PayPetal holds no money for the payout, and it's for an admin to
// pay by hand.
func (u *payoutUsecase) toManual(ctx context.Context, p *domain.Payout, msg string) {
	from := p.Status
	p.Status, p.ManualOnly, p.AutoError, p.RetryAt, p.WaitingOn = "due", true, msg, nil, ""
	if won, _ := u.payoutRepo.UpdateIfStatus(ctx, p, from); won {
		log.Printf("auto payout %s: handed over to be paid by hand", p.ID)
	}
}

// hold stops automatic attempts until an admin looks. Only for a PayPetal
// record that doesn't match the payout: retrying can't fix that, and
// releasing it could pay the wrong person.
func (u *payoutUsecase) hold(ctx context.Context, p *domain.Payout, msg string) {
	if p.Held && p.AutoError == msg {
		return
	}
	log.Printf("auto payout %s: needs an admin: %s", p.ID, msg)
	p.Held, p.AutoError = true, msg
	_, _ = u.payoutRepo.UpdateIfStatus(ctx, p, p.Status)
}

func (u *payoutUsecase) note(ctx context.Context, p *domain.Payout, msg string) {
	if p.AutoError != msg {
		p.AutoError = msg
		_, _ = u.payoutRepo.UpdateIfStatus(ctx, p, p.Status)
	}
}

func apiMessage(err error) string {
	var apiErr *paypetal.APIError
	if errors.As(err, &apiErr) && apiErr.Message != "" {
		return apiErr.Message
	}
	return err.Error()
}

// payoutWallet finds the payout sub-wallet, creating it the first time.
func (u *payoutUsecase) payoutWallet(ctx context.Context) (*paypetal.SubWallet, error) {
	u.walletMu.Lock()
	defer u.walletMu.Unlock()
	if u.subWalletID != "" {
		return &paypetal.SubWallet{ID: u.subWalletID, RecordID: u.subWalletRef}, nil
	}
	w, err := u.client.GetSubWallet(ctx, u.subWalletRef)
	if errors.Is(err, paypetal.ErrNotFound) {
		log.Printf("auto payout: creating payout sub-wallet %q", u.subWalletRef)
		w, err = u.client.CreateSubWallet(ctx, payoutWalletName, u.subWalletRef)
	}
	if err != nil {
		return nil, err
	}
	if w.ID == "" {
		return nil, errors.New("PayPetal returned a sub-wallet without an ID")
	}
	u.subWalletID = w.ID
	return w, nil
}

func (u *payoutUsecase) forgetWallet() {
	u.walletMu.Lock()
	u.subWalletID = ""
	u.walletMu.Unlock()
}

// walletBalance is the payout wallet's balance, read at most every
// balanceFresh so a pass over many waiting payouts asks PayPetal once.
func (u *payoutUsecase) walletBalance(ctx context.Context) (float64, bool) {
	u.walletMu.Lock()
	defer u.walletMu.Unlock()
	if !u.balanceAt.IsZero() && time.Since(u.balanceAt) < balanceFresh {
		return u.balance, true
	}
	w, err := u.client.GetSubWallet(ctx, u.subWalletRef)
	if err != nil {
		return 0, false
	}
	u.balance, u.balanceAt = w.Balance, time.Now()
	return u.balance, true
}

func (u *payoutUsecase) AutoPayouts(ctx context.Context) (*domain.AutoPayoutInfo, error) {
	if !u.autoEnabled() {
		return &domain.AutoPayoutInfo{Enabled: false}, nil
	}
	info := &domain.AutoPayoutInfo{Enabled: true, Reference: u.subWalletRef}
	if _, err := u.payoutWallet(ctx); err != nil {
		info.Error = "Couldn't reach the payout sub-wallet: " + apiMessage(err)
		return info, nil
	}
	w, err := u.client.GetSubWallet(ctx, u.subWalletRef)
	if err != nil {
		info.Error = "Couldn't read the payout sub-wallet: " + apiMessage(err)
		return info, nil
	}
	info.Balance, info.BankName, info.AccountNumber, info.AccountName = w.Balance, w.BankName, w.AccountNumber, w.AccountName
	return info, nil
}

func (u *payoutUsecase) RetryAuto(ctx context.Context, payoutID string) (*domain.Payout, error) {
	if !u.autoEnabled() {
		return nil, errors.New("automatic payouts are off")
	}
	u.pass.Lock()
	defer u.pass.Unlock()
	p, err := u.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, errors.New("payout not found")
	}
	switch p.Status {
	case "due":
		if p.ManualOnly && p.AutoRef != "" {
			// PayPetal holds nothing under the old reference any more (it
			// only became manual-only once that was confirmed), so start a
			// fresh one.
			p.AutoRound++
			p.AutoRef = ""
		}
		p.ManualOnly = false
	case "sending":
	default:
		return nil, fmt.Errorf("this payout is %s", p.Status)
	}
	from := p.Status
	p.AutoAttempts, p.RetryAt, p.WaitingOn, p.Held = 0, nil, "", false
	if won, err := u.payoutRepo.UpdateIfStatus(ctx, p, from); err != nil || !won {
		return nil, errors.New("this payout changed meanwhile — refresh and try again")
	}
	stepCtx, cancel := context.WithTimeout(ctx, autoStepTimeout)
	defer cancel()
	u.step(stepCtx, p)
	return u.payoutRepo.GetByID(ctx, payoutID)
}

func (u *payoutUsecase) PayByHand(ctx context.Context, payoutID string) (*domain.Payout, error) {
	u.pass.Lock()
	defer u.pass.Unlock()
	p, err := u.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, errors.New("payout not found")
	}
	switch p.Status {
	case "due":
		p.ManualOnly, p.AutoError, p.RetryAt, p.WaitingOn = true, "", nil, ""
		if won, err := u.payoutRepo.UpdateIfStatus(ctx, p, "due"); err != nil || !won {
			return nil, errors.New("this payout changed meanwhile — refresh and try again")
		}
		return p, nil
	case "stopping":
		return p, nil
	case "sending":
	default:
		return nil, fmt.Errorf("this payout is %s", p.Status)
	}

	tc, err := u.client.GetTrustCore(ctx, p.AutoRef)
	switch {
	case errors.Is(err, paypetal.ErrNotFound):
		u.toManual(ctx, p, "")
		return u.payoutRepo.GetByID(ctx, payoutID)
	case err != nil:
		return nil, fmt.Errorf("couldn't check with PayPetal: %w", err)
	}
	if problem := u.mismatch(ctx, p, tc); problem != "" {
		// Not provably ours, so don't touch it: hand the payout over and
		// leave the agreement for someone to look at in PayPetal.
		u.toManual(ctx, p, problem+" GigPurse left it alone.")
		return u.payoutRepo.GetByID(ctx, payoutID)
	}
	switch {
	case tc.PayoutStatus == "COMPLETED":
		u.markPaid(ctx, p, "sending", "PayPetal "+p.AutoRef, "paypetal")
		return nil, errors.New("PayPetal has already paid this one")
	case inFlight(tc.PayoutStatus):
		return nil, errors.New("the bank transfer is already on its way — wait for it to finish")
	case tc.RefundStatus == "COMPLETED":
		u.toManual(ctx, p, "")
		return u.payoutRepo.GetByID(ctx, payoutID)
	}
	if !inFlight(tc.RefundStatus) {
		if err := u.client.RefundTrustCore(ctx, p.AutoRef); err != nil && !paypetal.IsCode(err, "already_refunded") && !paypetal.IsCode(err, "refund_in_progress") {
			return nil, fmt.Errorf("PayPetal couldn't return the money to the sub-wallet: %s", apiMessage(err))
		}
	}
	p.Status, p.AutoError = "stopping", "Returning the money to the payout sub-wallet."
	if won, err := u.payoutRepo.UpdateIfStatus(ctx, p, "sending"); err != nil || !won {
		return nil, errors.New("this payout changed meanwhile — refresh and try again")
	}
	u.advance(ctx, p, false)
	return u.payoutRepo.GetByID(ctx, payoutID)
}
