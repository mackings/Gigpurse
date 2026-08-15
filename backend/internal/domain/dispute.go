package domain

import (
	"context"
	"time"
)

type Dispute struct {
	ID         string `json:"id" bson:"_id"`
	ContractID string `json:"contract_id" bson:"contract_id"`
	ClientID   string `json:"client_id" bson:"client_id"`
	MusicianID string `json:"musician_id" bson:"musician_id"`
	OpenedByID string `json:"opened_by_id" bson:"opened_by_id"`
	Reason     string `json:"reason" bson:"reason"`
	Status     string `json:"status" bson:"status"` // "open", "resolved", "closed"

	// ModeratorID is empty until a moderator/admin joins the dispute's chat
	// room — the two original parties can't message each other in that room
	// until this is set (see DisputeChatUsecase.SendMessage).
	ModeratorID string `json:"moderator_id,omitempty" bson:"moderator_id,omitempty"`

	Resolution string `json:"resolution,omitempty" bson:"resolution,omitempty"`
	// WinnerID is set only on disputes resolved before partial settlement
	// existed — kept so old resolved disputes still display correctly.
	// Current resolutions record TalentAmountNaira instead, which captures
	// full-refund (0), full-release (== funded amount), and partial (in
	// between) in one number rather than a binary winner.
	WinnerID string `json:"winner_id,omitempty" bson:"winner_id,omitempty"`
	// TalentAmountNaira is how much of the disputed funded amount the
	// moderator awarded the talent — 0 means a full refund to the client,
	// the full funded amount means a full release, anything in between is
	// a partial settlement (see DisputeUsecase.ResolveDispute for how that
	// gets executed, since PayPetal itself has no partial-settlement call).
	TalentAmountNaira float64 `json:"talent_amount,omitempty" bson:"talent_amount,omitempty"`

	// MilestoneID scopes a dispute to one specific milestone — set when a
	// single milestone withdrawal (not a whole contract ending) triggered
	// it, so resolution only touches that milestone rather than sweeping
	// every funded milestone on the contract.
	MilestoneID string `json:"milestone_id,omitempty" bson:"milestone_id,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`

	// ContractTitle is computed at query time only (never persisted) — the
	// admin dispute list shows this instead of the raw ContractID so a
	// moderator can tell disputes apart at a glance.
	ContractTitle string `json:"contract_title,omitempty" bson:"-"`
}

type DisputeRepository interface {
	Create(ctx context.Context, dispute *Dispute) error
	GetByID(ctx context.Context, id string) (*Dispute, error)
	Update(ctx context.Context, dispute *Dispute) error
	List(ctx context.Context, status string) ([]*Dispute, error)
	ListForUser(ctx context.Context, userID string) ([]*Dispute, error)
}

type DisputeUsecase interface {
	OpenDispute(ctx context.Context, userID, contractID, reason string) (*Dispute, error)
	ListUserDisputes(ctx context.Context, userID string) ([]*Dispute, error)
	ListAllDisputes(ctx context.Context, status string) ([]*Dispute, error)
	GetDispute(ctx context.Context, requesterID, disputeID string) (*Dispute, error)
	// ResolveDispute takes how much of the disputed funded amount the
	// talent gets — 0 refunds the client in full, the full funded amount
	// releases it all to the talent, anything in between refunds the
	// client in full immediately (the only settlement PayPetal itself
	// supports) and creates a new "Dispute settlement" milestone for that
	// amount that the client must separately fund. Scope is the dispute's
	// MilestoneID if set, else every currently `funded` milestone on the
	// contract.
	ResolveDispute(ctx context.Context, resolverID, disputeID, resolution string, talentAmountNaira float64) (*Dispute, error)

	// EndContract lets either party end a contract outright. Any
	// not-yet-funded milestone just cancels. Any `funded` milestone means
	// real money is on the line, so instead the contract goes "disputed"
	// and a dispute opens automatically (scoped to every funded milestone)
	// rather than letting the money hang in limbo.
	EndContract(ctx context.Context, userID, contractID string) (*Contract, error)
	// CancelMilestone is EndContract's single-milestone counterpart —
	// pulling one milestone instead of the whole contract. `accepted`
	// cancels cleanly; `funded` opens a dispute scoped to just this one.
	CancelMilestone(ctx context.Context, userID, contractID, milestoneID string) (*Milestone, error)
	// HasOutstandingSettlement reports whether this client has an unfunded
	// "Dispute settlement" milestone waiting on them anywhere — used to
	// block new activity (posting jobs, hiring) until they pay what a
	// moderator already ordered.
	HasOutstandingSettlement(ctx context.Context, clientID string) (bool, error)

	// JoinDispute lets a moderator/admin attach themselves to a dispute's
	// chat room — this is what unblocks messaging between the two original
	// parties, and posts an automatic system message announcing it.
	JoinDispute(ctx context.Context, moderatorID, disputeID string) (*Dispute, error)
	SendDisputeMessage(ctx context.Context, senderID, disputeID, content, attachmentURL, attachmentType, mentionedUserID string) (*ChatMessage, error)
	ListDisputeMessages(ctx context.Context, requesterID, disputeID string) ([]*ChatMessage, error)
}
