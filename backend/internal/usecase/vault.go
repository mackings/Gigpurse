package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gigpurse/internal/domain"
)

// ErrReleaseCodeRequired means this money is in TrustVault, where nothing can
// move — release or refund — without the 6-digit code PayPetal emails to the
// client who paid. Exported so handlers can answer "release_code_required".
var ErrReleaseCodeRequired = errors.New("this payment needs the 6-digit code emailed to the client before it can be released")

// errLegacyPayment is any record from before TrustVault. GigPurse no longer
// talks to the product those used.
var errLegacyPayment = errors.New("this payment was made on an older system — contact support")

const (
	// nudgeReleaseAfter is how long a client gets after a release request
	// before reminders start; vaultNudgeEvery is how often they repeat.
	nudgeReleaseAfter = 48 * time.Hour
	vaultNudgeEvery   = 24 * time.Hour
)

// vaultAgreementFor loads the agreement behind a milestone and checks the
// caller is the client who paid it — the only person PayPetal sends codes to.
func (u *milestoneUsecase) vaultAgreementFor(ctx context.Context, userID, referenceID string) (*domain.EscrowAgreement, *domain.Milestone, error) {
	agreement, err := u.escrowRepo.GetByReference(ctx, referenceID)
	if err != nil {
		return nil, nil, errors.New("payment not found")
	}
	if agreement.InitiatorUserID != userID {
		return nil, nil, errors.New("unauthorized: only the client who paid can do this")
	}
	if !agreement.IsVault() {
		return nil, nil, errLegacyPayment
	}
	milestone, err := u.milestoneRepo.GetByID(ctx, agreement.ScopeID)
	if err != nil {
		return nil, nil, fmt.Errorf("milestone not found: %w", err)
	}
	return agreement, milestone, nil
}

func (u *milestoneUsecase) SendReleaseCode(ctx context.Context, userID, referenceID string) error {
	_, milestone, err := u.vaultAgreementFor(ctx, userID, referenceID)
	if err != nil {
		return err
	}
	if err := codeUsable(milestone); err != nil {
		return err
	}
	if err := u.client.SendVaultReleaseCode(ctx, referenceID); err != nil {
		return fmt.Errorf("couldn't send the code: %w", err)
	}
	return nil
}

func (u *milestoneUsecase) SendMilestoneReleaseCode(ctx context.Context, contractID, milestoneID, userID string) error {
	_, milestone, _, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return err
	}
	return u.SendReleaseCode(ctx, userID, milestone.EscrowReference)
}

// codeUsable rejects a code request for anything but money the client may
// currently act on: a funded milestone, or one a moderator has ruled on.
func codeUsable(m *domain.Milestone) error {
	switch m.Status {
	case "funded", "settling":
		return nil
	case "disputed":
		return errors.New("this payment is under dispute — a moderator has to settle it")
	default:
		return fmt.Errorf("this milestone is %s", m.Status)
	}
}

// Release pays out a milestone with the client's code: to the talent if it's
// simply funded, or however the moderator ruled if it's settling.
func (u *milestoneUsecase) Release(ctx context.Context, contractID, milestoneID, userID, code string) (*domain.Milestone, error) {
	_, loaded, _, err := u.loadForTransition(ctx, contractID, milestoneID, userID)
	if err != nil {
		return nil, err
	}
	agreement, milestone, err := u.vaultAgreementFor(ctx, userID, loaded.EscrowReference)
	if err != nil {
		return nil, err
	}
	if err := codeUsable(milestone); err != nil {
		return nil, err
	}
	award := agreement.AmountNaira
	if milestone.Status == "settling" && milestone.SettlementTalent != nil {
		award = *milestone.SettlementTalent
	}
	return u.settleVault(ctx, agreement, milestone, award, code)
}

// settleVault moves everything held to GigPurse's wallet with the client's
// code, and records what GigPurse then owes: the talent's award, and whatever
// part goes back to the client. A full award keeps GigPurse's fee; anything
// less returns the fee to the client along with the rest.
//
// What's owed is written down as "pending" before PayPetal is called, so a
// crash or a lost response can't leave released money with no record of who
// it belongs to — ReconcileVault finishes or undoes it.
func (u *milestoneUsecase) settleVault(ctx context.Context, agreement *domain.EscrowAgreement, milestone *domain.Milestone, award float64, code string) (*domain.Milestone, error) {
	if code == "" {
		return nil, ErrReleaseCodeRequired
	}
	start, final := milestone.Status, "released"
	refund := agreement.RefundTotal() - award
	if award >= agreement.AmountNaira-settlementEpsilon {
		award, refund = agreement.AmountNaira, 0
	} else if award <= settlementEpsilon {
		award, final = 0, "refunded"
	}

	won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, start, final)
	if err != nil {
		return nil, err
	}
	if !won {
		return nil, errors.New("this payment is already being settled")
	}
	revert := func() { _, _ = u.milestoneRepo.CompareAndSwapStatus(ctx, milestone.ID, final, start) }

	var owed []*domain.Payout
	add := func(kind, userID string, amount float64) {
		if amount > settlementEpsilon {
			owed = append(owed, &domain.Payout{
				ID: agreement.ReferenceID + ":" + kind, ReferenceID: agreement.ReferenceID,
				MilestoneID: milestone.ID, ContractID: milestone.ContractID, UserID: userID,
				Kind: kind, Title: milestone.Title, Amount: amount, RevertTo: start,
			})
		}
	}
	add(domain.PayoutTalent, agreement.CounterpartyUserID, award)
	add(domain.PayoutRefund, agreement.InitiatorUserID, refund)
	if err := u.payouts.ReplacePending(ctx, agreement.ReferenceID, owed); err != nil {
		revert()
		return nil, fmt.Errorf("couldn't record the payout: %w", err)
	}

	if err := u.client.ReleaseVaultAgreement(ctx, agreement.ReferenceID, code); err != nil {
		// Whatever the error says, ask PayPetal what actually happened: a
		// timeout may have gone through, and a retry of one that did comes
		// back as "already completed".
		state, stateErr := u.client.GetVaultAgreement(ctx, agreement.ReferenceID)
		switch {
		case stateErr != nil:
			log.Printf("vault %s: release outcome unknown (%v; %v) — leaving for the reconciler", agreement.ReferenceID, err, stateErr)
			return nil, errors.New("PayPetal hasn't confirmed this release yet — we're checking, please don't try again")
		case state.Status != "COMPLETED":
			_ = u.payouts.ReplacePending(ctx, agreement.ReferenceID, nil)
			revert()
			return nil, fmt.Errorf("couldn't release the payment: %w", err)
		}
	}
	u.confirmVaultRelease(ctx, agreement, owed)
	milestone.Status = final
	return milestone, nil
}

// confirmVaultRelease runs once PayPetal has released the money. Each row
// announces itself only if this call is what moved it to "due", so the direct
// path and the reconciler can both reach here safely.
func (u *milestoneUsecase) confirmVaultRelease(ctx context.Context, agreement *domain.EscrowAgreement, owed []*domain.Payout) {
	for _, p := range owed {
		won, err := u.payouts.CompareAndSwapStatus(ctx, p.ID, "pending", "due")
		if err != nil || !won {
			if err != nil {
				log.Printf("vault %s: marking %s due failed: %v", agreement.ReferenceID, p.ID, err)
			}
			continue
		}
		if p.Kind == domain.PayoutTalent {
			agreement.PayoutStatus = "MANUAL_DUE"
			_ = u.walletRepo.AddTransaction(ctx, &domain.Transaction{
				UserID: agreement.InitiatorUserID, Type: "escrow_release", Amount: p.Amount,
				Description: fmt.Sprintf("Payment released: %s", p.Title), Reference: agreement.ReferenceID,
			})
			u.notify(ctx, p.UserID, "Payment released",
				fmt.Sprintf("%s for '%s' has been released. GigPurse is sending it to your bank account — we'll tell you when it's on its way.", formatNaira(p.Amount), p.Title), p.ContractID)
		} else {
			agreement.RefundStatus = "MANUAL_DUE"
			u.notify(ctx, p.UserID, "Refund approved",
				fmt.Sprintf("Your refund of %s for '%s' is approved. GigPurse is sending it to your bank account — we'll tell you when it's on its way.", formatNaira(p.Amount), p.Title), p.ContractID)
			if agreement.PayoutStatus == "NONE" {
				u.notify(ctx, agreement.CounterpartyUserID, "Milestone payment refunded",
					fmt.Sprintf("The payment for milestone '%s' is being returned to the client.", p.Title), p.ContractID)
			}
		}
	}
	_ = u.escrowRepo.Update(ctx, agreement)
}

// ReconcileVault settles releases whose outcome PayPetal hadn't confirmed
// when the client entered their code.
func (u *milestoneUsecase) ReconcileVault(ctx context.Context) {
	pending, err := u.payouts.ListByStatus(ctx, "pending")
	if err != nil {
		return
	}
	byRef := map[string][]*domain.Payout{}
	for _, p := range pending {
		// A release in progress right now is still "pending" for a moment.
		if time.Since(p.CreatedAt) > time.Minute {
			byRef[p.ReferenceID] = append(byRef[p.ReferenceID], p)
		}
	}
	for ref, owed := range byRef {
		agreement, err := u.escrowRepo.GetByReference(ctx, ref)
		if err != nil {
			continue
		}
		state, err := u.client.GetVaultAgreement(ctx, ref)
		if err != nil {
			continue
		}
		if state.Status == "COMPLETED" {
			u.confirmVaultRelease(ctx, agreement, owed)
			continue
		}
		log.Printf("vault %s: release never happened, reopening", ref)
		if err := u.payouts.ReplacePending(ctx, ref, nil); err != nil {
			continue
		}
		for _, from := range []string{"released", "refunded"} {
			if won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, owed[0].MilestoneID, from, owed[0].RevertTo); err == nil && won {
				break
			}
		}
	}
}

// BeginVaultSettlement records a moderator's ruling on a disputed TrustVault
// milestone. The money can't move until the client enters their code, so the
// milestone waits as "settling" — and the client can't start anything new
// until they do (see HasOutstandingSettlement).
func (u *milestoneUsecase) BeginVaultSettlement(ctx context.Context, milestoneID string, talentAward float64) error {
	won, err := u.milestoneRepo.CompareAndSwapStatus(ctx, milestoneID, "disputed", "settling")
	if err != nil {
		return err
	}
	if !won {
		return errors.New("milestone isn't under dispute")
	}
	milestone, err := u.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		return err
	}
	milestone.SettlementTalent = &talentAward
	if err := u.milestoneRepo.Update(ctx, milestone); err != nil {
		return err
	}
	if contract, err := u.contractRepo.GetByID(ctx, milestone.ContractID); err == nil {
		u.notify(ctx, contract.ClientID, "Confirm the dispute ruling",
			fmt.Sprintf("A moderator ruled on '%s': %s to the talent. Open the contract and enter the code PayPetal emails you to settle it — you can't start new gigs until you do.", milestone.Title, formatNaira(talentAward)), contract.ID)
		u.notify(ctx, contract.MusicianID, "Dispute ruled",
			fmt.Sprintf("A moderator awarded you %s for '%s'. The client has to confirm it with a code before it can be paid.", formatNaira(talentAward), milestone.Title), contract.ID)
	}
	return nil
}

// StartReleaseNudgeScanner runs for the lifetime of ctx, reminding clients
// about release requests they've left unanswered.
func (u *milestoneUsecase) StartReleaseNudgeScanner(ctx context.Context, checkInterval time.Duration) {
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.nudgeOverdueReleases(ctx)
			}
		}
	}()
}

func (u *milestoneUsecase) nudgeOverdueReleases(ctx context.Context) {
	funded, err := u.milestoneRepo.ListByStatus(ctx, "funded")
	if err != nil {
		return
	}
	for _, m := range funded {
		if m.EscrowProvider != domain.ProviderVault || m.ReleaseRequestedAt == nil || time.Since(*m.ReleaseRequestedAt) < nudgeReleaseAfter {
			continue
		}
		if contract, err := u.contractRepo.GetByID(ctx, m.ContractID); err == nil {
			u.nudgeVaultRelease(ctx, contract, m)
		}
	}
}

// nudgeVaultRelease: nobody but the client can release TrustVault money, so
// all the system can do is keep asking them and tell the talent how to
// escalate.
func (u *milestoneUsecase) nudgeVaultRelease(ctx context.Context, contract *domain.Contract, m *domain.Milestone) {
	now := time.Now()
	if m.ReleaseNudgedAt != nil && now.Sub(*m.ReleaseNudgedAt) < vaultNudgeEvery {
		return
	}
	first := m.ReleaseNudgedAt == nil
	m.ReleaseNudgedAt = &now
	if err := u.milestoneRepo.Update(ctx, m); err != nil {
		return
	}
	u.notify(ctx, contract.ClientID, "Please release this payment",
		fmt.Sprintf("The talent asked for '%s' over 48 hours ago. Release it with the code PayPetal emails you, or open a dispute.", m.Title), contract.ID)
	if first {
		u.notify(ctx, contract.MusicianID, "The client hasn't released your payment",
			fmt.Sprintf("It's been 48 hours since you asked for '%s'. We've reminded the client. You can open a dispute and a moderator will step in.", m.Title), contract.ID)
	}
}
