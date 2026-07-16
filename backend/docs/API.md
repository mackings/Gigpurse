# Gigpurse Backend API Documentation

Base URL for local development: `http://localhost:8080`

Authentication uses JWT bearer tokens:

```http
Authorization: Bearer <token>
```

All HTTP JSON responses now use the same envelope shape.

Success response:

```json
{
  "success": true,
  "status": "success",
  "status_code": 200,
  "message": "operation completed successfully",
  "data": {}
}
```

Error response:

```json
{
  "success": false,
  "status": "error",
  "status_code": 400,
  "message": "invalid request body",
  "error": {
    "code": "invalid_request_body",
    "message": "invalid request body"
  }
}
```

Common status codes:

| Code | Meaning |
| --- | --- |
| `200 OK` | Request succeeded. |
| `201 Created` | Resource created. |
| `400 Bad Request` | Invalid body, missing required field, invalid workflow state, or validation failure. |
| `401 Unauthorized` | Missing, malformed, invalid, or expired token. |
| `403 Forbidden` | Authenticated user does not have the required role or ownership. |
| `404 Not Found` | Resource was not found. |
| `405 Method Not Allowed` | Endpoint exists but does not support the HTTP method. |
| `500 Internal Server Error` | Storage or unexpected server-side failure. |

Wallet and milestone-based escrow are real, authenticated, and persisted in
MongoDB — see the Wallet and Milestones sections. What's still pending is
connecting a real external payment processor (deposits/withdrawals don't
move real money yet).

## Health

### `GET /`

Checks service availability.

Auth: not required

Response `200`:

```json
{
  "status": "online",
  "service": "gigpurse-backend"
}
```

Status codes: `200`, `404`.

## Auth

### `POST /auth/signup`

Creates a user account.

Auth: not required

Required body:

```json
{
  "email": "client@example.com",
  "password": "password123",
  "role": "client",
  "name": "Demo Client",
  "accepted_terms": true
}
```

`accepted_terms` must be `true` — signup is rejected otherwise. The acceptance
timestamp is stored on the user as `terms_accepted_at`.

Roles: `client`, `musician`. `admin` is allowed only when `ALLOW_ADMIN_SIGNUP=true`;
`moderator` (scoped access — disputes only, see Disputes And Customer Service)
only when `ALLOW_MODERATOR_SIGNUP=true`.

Response `201`. Signup also sends a 6-digit email verification code. Login is blocked until the email is verified.

```json
{
  "success": true,
  "status": "success",
  "status_code": 201,
  "message": "signup successful. verify your email before login",
  "data": {
    "id": "usr_1",
    "email": "client@example.com",
    "email_verified": false,
    "role": "client",
    "name": "Demo Client"
  }
}
```

Status codes: `201`, `400`, `405`, `500`.

### `POST /auth/login`

Logs in a user and returns a JWT.

Auth: not required

Required body:

```json
{
  "email": "client@example.com",
  "password": "password123"
}
```

Response `200`:

```json
{
  "success": true,
  "status": "success",
  "status_code": 200,
  "message": "login successful",
  "data": {
    "token": "<jwt>",
    "user": {
      "id": "usr_1",
      "email": "client@example.com",
      "role": "client",
      "name": "Demo Client"
    }
  }
}
```

Status codes: `200`, `400`, `401`, `405`.

### `POST /auth/email-verification/resend`

Sends or resends the 6-digit email verification code.

Auth: not required

Required body:

```json
{
  "email": "client@example.com"
}
```

Response `200`:

```json
{
  "success": true,
  "status": "success",
  "status_code": 200,
  "message": "if the email exists and is unverified, a verification message has been sent"
}
```

Status codes: `200`, `400`, `405`.

### `POST /auth/email-verification/confirm`

Verifies a user email using the 6-digit code sent after signup.

Auth: not required

Required body:

```json
{
  "email": "client@example.com",
  "code": "123456"
}
```

Response `200`:

```json
{
  "success": true,
  "status": "success",
  "status_code": 200,
  "message": "email verified successfully"
}
```

Status codes: `200`, `400`, `405`.

Email delivery tries providers in this order, using the first one that's
fully configured via env vars:

1. **Resend** — `RESEND_API_KEY`, `RESEND_FROM_EMAIL` (e.g. `"House of GLAME <noreply@example.com>"`)
2. **Mailjet** — `MAILJET_API_KEY`, `MAILJET_API_SECRET`, `MAILJET_FROM_EMAIL`, `MAILJET_FROM_NAME` (optional, defaults to "Gigpurse")
3. **SMTP** — `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`

If none are configured, the backend logs the email content (including the
verification code / reset token) to the server outbox instead of
delivering it — this is expected for local development.

### `POST /auth/password-reset/request`

Requests a password reset email. The current implementation logs the token to the email outbox.

Auth: not required

Required body:

```json
{
  "email": "client@example.com"
}
```

Response `200`:

```json
{
  "message": "if the email exists, a password reset message has been sent"
}
```

Status codes: `200`, `400`, `405`.

### `POST /auth/password-reset/confirm`

Resets a password using a valid reset token.

Auth: not required

Required body:

```json
{
  "token": "reset-token-from-email",
  "new_password": "newPassword123"
}
```

Response `200`:

```json
{
  "message": "password reset successfully"
}
```

Status codes: `200`, `400`, `405`, `500`.

## Profiles And Musician Search

### `GET /users/profile`

Returns the authenticated user profile.

Auth: required

Response `200`:

```json
{
  "id": "usr_1",
  "email": "client@example.com",
  "role": "client",
  "name": "Demo Client",
  "bio": "Looking for reliable session musicians",
  "location": "Lagos"
}
```

Status codes: `200`, `401`, `404`, `405`.

### `PUT /users/profile`

Updates the authenticated user profile.

Auth: required

Client body:

```json
{
  "name": "Demo Client",
  "bio": "Looking for reliable session musicians",
  "location": "Lagos",
  "client_profile": {
    "company_name": "Gigpurse Events"
  }
}
```

Musician body:

```json
{
  "name": "Demo Musician",
  "bio": "Guitarist and producer",
  "location": "Lagos",
  "musician_profile": {
    "stage_name": "Demo Strings",
    "instruments": ["Guitar", "Bass"],
    "genres": ["Afrobeats", "Highlife"],
    "experience_years": 7,
    "price_min": 200,
    "price_max": 600,
    "availability": ["Weekends", "Evenings"],
    "social_links": {
      "instagram": "https://instagram.com/demostrings",
      "youtube": "https://youtube.com/@demostrings",
      "spotify": "https://open.spotify.com/artist/demostrings"
    },
    "intro_video_url": "https://youtube.com/watch?v=demo",
    "portfolio": [
      {
        "title": "Live Session",
        "description": "Recorded guitar session",
        "url": "https://example.com/session.mp4",
        "media_type": "video",
        "thumbnail_url": "https://example.com/session-thumb.jpg",
        "is_featured": true,
        "order": 0
      }
    ]
  }
}
```

`genres` and `instruments` are arrays (a musician can list more than one). `price_min`,
`price_max`, `availability`, `social_links`, and `intro_video_url` are all optional.
Portfolio item fields `media_type` (`video`, `audio`, or `image`), `external_url`,
`thumbnail_url`, `is_featured`, and `order` are all optional.

Response `200`: updated `User`.

Status codes: `200`, `400`, `401`, `405`, `500`.

### `GET /musicians`

Searches musician profiles.

Auth: not required by backend.

Query parameters:

| Name | Description |
| --- | --- |
| `genre` | Case-insensitive genre filter; matches if any of the musician's `genres` contains it. |
| `instrument` | Case-insensitive instrument filter; matches if any of the musician's `instruments` contains it. |
| `location` | Case-insensitive location filter. |
| `min_exp` | Minimum years of experience. |
| `sort_by` | `experience`, `newest`, or `rating`. |
| `sort_order` | `asc` or `desc`. |

Every result includes computed `average_rating` and `total_reviews` fields
(via an aggregation against the reviews collection), so the frontend can
render a rating per card without a follow-up `GET /reviews/average` call per
musician.

Example:

```http
GET /musicians?genre=Afrobeats&instrument=Guitar&location=Lagos&min_exp=3&sort_by=rating&sort_order=desc
```

Response `200`:

```json
[
  {
    "id": "usr_2",
    "email": "musician@example.com",
    "role": "musician",
    "name": "Demo Musician",
    "location": "Lagos",
    "musician_profile": {
      "stage_name": "Demo Strings",
      "instruments": ["Guitar", "Bass"],
      "genres": ["Afrobeats", "Highlife"],
      "experience_years": 7
    },
    "average_rating": 4.8,
    "total_reviews": 12
  }
]
```

Status codes: `200`, `405`, `500`.

### `GET /musicians/{id}`

Returns a single musician's public profile by ID.

Auth: not required by backend.

Response `200`: a single `User` (same shape as `GET /musicians` list entries), plus two
public trust-signal fields computed at request time: `completed_contracts` (count of this
musician's contracts with `status: "completed"`) and `total_earned` (sum of those
contracts' `price`) — shown on the public profile the way marketplaces like Upwork surface
"$X earned". Returns `404` if the ID doesn't exist or doesn't belong to a musician.

Status codes: `200`, `404`.

### `GET /users/{id}`

Returns a minimal, non-sensitive projection of any user by ID, regardless of
role — `{"id", "name", "role", "location", "created_at", "client_profile",
"status"}`. Used to resolve display names (e.g. a chat partner) and to
render the "About the client" panel on a job's detail page, without
exposing full profile data (no email, no musician-specific fields).

`status` is the presence status as seen by someone else — one of `"online"`,
`"offline"`, `"disabled"`. It's never `"hidden"`: a user who has enabled
`hide_presence` (see below) shows as `"offline"` to everyone but themselves,
which is the entire point of that setting.

Auth: required.

Status codes: `200`, `401`, `404`.

### `PUT /users/account-status`

Self-service settings toggle — always acts on the authenticated caller,
never an arbitrary user ID. Both flags default `false` and are fully
reversible by the account owner; `disabled` is a "pause my account" toggle,
not a suspension — the owner stays able to log in and re-enable it
themselves at any time.

Auth: required.

Body:

```json
{ "hide_presence": true, "disabled": false }
```

- `hide_presence` — when `true`, the account always shows as `"offline"` to
  others regardless of actual connection state.
- `disabled` — when `true`, the account shows as `"disabled"` to others,
  is excluded from `GET /musicians` results (if the account is a
  musician), and can't receive new chat messages (`POST /chats` to this
  user returns `400`).

Response `200`: the updated `User`.

Status codes: `200`, `400`, `401`.

## Jobs And Applications

### `POST /jobs`

Creates a job listing. New jobs start `status: "pending_funding"` — they are
invisible to talent (excluded from `status=open` listings, recommendations,
and applications) until the client funds escrow via `POST /jobs/fund`. This
guarantees that any job talent can see and apply to already has its budget
held in escrow.

Auth: required, role `client`.

Required body:

```json
{
  "title": "Afrobeats guitar session",
  "description": "Need guitar for a studio session",
  "instrument": "Guitar",
  "genre": "Afrobeats",
  "location": "Lagos",
  "budget": 500,
  "experience_level": "intermediate",
  "duration": "less_than_1_week",
  "project_type": "one_time",
  "skills": ["Guitar", "Session recording"]
}
```

`experience_level`, `duration`, `project_type`, and `skills` are all
optional. `experience_level` is one of `entry`/`intermediate`/`expert`.
`duration` is one of `less_than_1_week`/`1_to_2_weeks`/`1_to_4_weeks`/
`1_to_3_months`/`3_plus_months`. `project_type` is `one_time` or `ongoing`.

Response `201`:

```json
{
  "id": "job_1",
  "client_id": "usr_1",
  "title": "Afrobeats guitar session",
  "description": "Need guitar for a studio session",
  "budget": 500,
  "instrument": "Guitar",
  "genre": "Afrobeats",
  "location": "Lagos",
  "status": "pending_funding",
  "escrow_funded": false
}
```

Status codes: `201`, `400`, `401`, `403`, `405`.

### `POST /jobs/fund`

Funds a `pending_funding` job's escrow from the client's wallet balance
(`GET /wallet` / `POST /wallet/deposit`) and flips it to `status: "open"`,
making it visible to talent. Fails with `400` if the wallet balance is
below the job's budget, or if the job isn't currently `pending_funding`
(already funded, or not owned by the caller).

Auth: required, role `client`, must be the job's creator.

Body: `{"job_id": "..."}`.

Response `200`: the updated `Job`, with `status: "open"`,
`escrow_funded: true`, and `escrow_amount` set to the budget that was held.

Status codes: `200`, `400`, `401`, `403`.

### `GET /jobs`

Lists jobs or fetches one job by ID.

Auth: not required by backend.

Query parameters:

| Name | Description |
| --- | --- |
| `id` | If present, returns one job (with the full `client` trust-stats object and `application_count` populated — see below). |
| `query` | Free-text search — case-insensitive substring match against title or description. |
| `status` | `pending_funding`, `open`, `active`, `completed`, `disputed`. |
| `genre` | Genre filter. |
| `instrument` | Instrument filter. |
| `location` | Location filter. |
| `min_budget` | Minimum budget. |
| `max_budget` | Maximum budget. |
| `sort_by` | `newest`, `budget`, `applications`, `popularity`, `relevance`. |
| `sort_order` | `asc` or `desc`. |
| `max_applications` | Only jobs with application count less than or equal to this value. |
| `client_id` | Only jobs posted by this client. Useful for a client's own job dashboard (includes their own `pending_funding` jobs, so they can complete funding). |

Every job in a list response carries `application_count` (real proposal
count) and lightweight `client_rating`/`client_review_count`. A single-job
fetch (`?id=`) additionally carries a full `client` object:

```json
{
  "id": "job_1",
  "status": "open",
  "title": "Afrobeats guitar session",
  "budget": 500,
  "escrow_funded": true,
  "escrow_amount": 500,
  "application_count": 3,
  "client_rating": 4.8,
  "client_review_count": 12,
  "client": {
    "name": "Demo Client",
    "company_name": "Gigpurse Events",
    "location": "Lagos",
    "member_since": "2026-01-04T10:00:00Z",
    "rating": 4.8,
    "review_count": 12,
    "jobs_posted": 6,
    "open_jobs": 2,
    "hire_rate": 66.7,
    "total_spent": 2100,
    "recent_hires": [
      { "musician_name": "Demo Musician", "job_title": "Wedding guitarist", "status": "completed", "date": "2026-06-01T00:00:00Z" }
    ]
  }
}
```

Status codes: `200`, `404`, `405`, `500`.

### `GET /jobs/recommended`

Returns personalized open gig recommendations for a musician — defaults to
the musician's own genres/instruments/location, but any of `query`, `genre`,
`instrument`, `location`, `min_budget`, `max_budget` passed explicitly
narrows the personalized result instead (used by the "Best matches" tab's
search bar and filters).

Auth: required, role `musician`.

Query parameters: `limit` (defaults to `10`, max practical limit `20`), plus
the same `query`/`genre`/`instrument`/`location`/`min_budget`/`max_budget`
params as `GET /jobs`.

Response `200`: array of `Job`.

Status codes: `200`, `400`, `401`, `403`, `405`.

### `POST /jobs/save` / `POST /jobs/unsave`

Saves or removes a job from the musician's saved-jobs list, for later
review — a saved job isn't filtered by status, so one that's since been
filled by another musician still shows up (as "Closed" in the UI) rather
than silently disappearing.

Auth: required, role `musician`.

Body: `{"job_id": "..."}`.

Status codes: `200`, `400`, `401`, `403`.

### `GET /jobs/saved`

Returns the musician's saved jobs (full `Job` objects, any status).

Auth: required, role `musician`.

Status codes: `200`, `401`, `403`, `500`.

### `GET /jobs/mine`

Returns musician jobs grouped by status.

Auth: required, role `musician`.

Query parameters:

| Name | Description |
| --- | --- |
| `status` | `pending`, `active`, or `completed`. Defaults to `pending`. |

Response `200`: array of `Job`.

Status codes: `200`, `401`, `403`, `405`, `500`.

### `POST /jobs/apply`

Submits a musician application to an open job.

Auth: required, role `musician`.

Required body:

```json
{
  "job_id": "job_1",
  "proposal": "I can deliver a clean session.",
  "price_bid": 450
}
```

Response `201`:

```json
{
  "id": "app_1",
  "job_id": "job_1",
  "musician_id": "usr_2",
  "proposal": "I can deliver a clean session.",
  "price_bid": 450,
  "status": "pending"
}
```

Status codes: `201`, `400`, `401`, `403`, `405`.

### `GET /jobs/applications`

Lists applications.

Auth: required.

Modes:

| Query | Behavior |
| --- | --- |
| `job_id=<id>` | Client who owns the job gets applications for that job. |
| no `job_id` | Musician gets their own applications. |

Response `200`: array of `JobApplication`.

Status codes: `200`, `400`, `401`, `403`, `404`, `405`, `500`.

### `POST /jobs/applications/accept`

Accepts an application, marks the job active, rejects other pending applications, and creates a contract.

Auth: required, role `client`, must own the job.

Required body:

```json
{
  "application_id": "app_1"
}
```

Response `200`:

```json
{
  "message": "application accepted successfully, job is now active"
}
```

Status codes: `200`, `400`, `401`, `403`, `405`, `500`.

## Chat

### `POST /chats`

Sends a chat message. External payment/contact terms are censored. Fails
with `400` if the recipient has disabled their account
(`PUT /users/account-status`).

Auth: required.

Required body:

```json
{
  "recv_id": "usr_2",
  "content": "Can we talk on WhatsApp or pay by Paypal?"
}
```

Response `201`:

```json
{
  "id": "msg_1",
  "sender_id": "usr_1",
  "recv_id": "usr_2",
  "content": "Can we talk on ******** or pay by ******?",
  "timestamp": "2026-06-19T18:58:33Z"
}
```

Status codes: `201`, `400`, `401`, `405`.

### `GET /chats/history`

Returns message history between the authenticated user and another user.

Auth: required.

Query: `user_id=<other_user_id>`

Response `200`: array of `ChatMessage`.

Status codes: `200`, `400`, `401`, `405`, `500`.

### `GET /chats/recent`

Returns latest message per chat partner.

Auth: required.

Response `200`: array of `ChatMessage`.

Status codes: `200`, `401`, `405`, `500`.

### `GET /ws?token=<jwt>`

Single realtime socket per user. Carries both live chat messages and
notification pushes — the frontend only ever needs one persistent
connection, opened once for the session (not per conversation).

Auth: JWT in `token` query parameter or `Authorization` header. Opening a
second connection for the same user closes the first.

Client → server message (sends a chat message):

```json
{
  "recv_id": "usr_2",
  "content": "Realtime hello"
}
```

Server → client frames are always wrapped in an envelope so the client can
dispatch by `type`:

```json
{ "type": "chat_message", "data": { "...": "ChatMessage" } }
{ "type": "notification", "data": { "...": "Notification" } }
{ "type": "error", "data": "invalid message format" }
```

`chat_message` is sent to the sender (as a send confirmation/echo) and to
the receiver if they're online. `notification` is pushed to a user
whenever any endpoint creates a `Notification` for them (job application
accepted, contract completed, dispute opened/resolved, review received,
etc.) — no polling required on the client.

Handshake/error status codes before upgrade: `101 Switching Protocols`, `401`.

## Contracts And Direct Hire

### `GET /contracts`

Lists contracts for the authenticated user, or fetches one contract by ID.

Auth: required.

Query parameters:

| Name | Description |
| --- | --- |
| `id` | Optional contract ID. If present, requester must be participant or admin. |

Response `200`:

```json
[
  {
    "id": "con_1",
    "job_id": "job_1",
    "client_id": "usr_1",
    "musician_id": "usr_2",
    "title": "Afrobeats guitar session",
    "price": 450,
    "source": "job",
    "status": "active"
  }
]
```

Status codes: `200`, `401`, `403`, `404`, `405`, `500`.

### `POST /contracts/complete`

Marks an active contract completed. Only the client can complete it.

Auth: required, contract client only.

Required body:

```json
{
  "contract_id": "con_1"
}
```

Response `200`:

```json
{
  "message": "contract marked completed successfully"
}
```

Status codes: `200`, `401`, `403`, `405`, `500`.

Booking (direct hire) requests support negotiation: either side can counter
the current offer (price, and optionally title/description/location/date)
until one side accepts or declines. `proposed_by` is whoever made the
current offer — only the *other* participant may accept, decline, or
counter it; the proposer must wait for a response. `history` is the full
list of offers made, oldest first. A notification fires on every propose/
counter/accept/decline, linking into that conversation's chat
(`/messages?with={other_user_id}&booking={request_id}`) so both sides can
negotiate and chat in the same place.

### `POST /direct-hires`

Creates a direct hire request — either a client proposing to a musician, or
a musician proposing to a client (the initial offer — `proposed_by` is set
to whoever is authenticated and calling this).

Auth: required, role `client` or `musician`.

Required body:

```json
{
  "target_user_id": "usr_2",
  "title": "Private acoustic set",
  "description": "Direct hire for a private event",
  "location": "Lekki, Lagos",
  "event_date": "2026-08-01T18:00:00Z",
  "price": 300
}
```

`target_user_id` is the other party — a musician if you're a client, or a
client if you're a musician. `musician_id` is still accepted as an alias for
`target_user_id` for backward compatibility with existing client-initiated
callers. `location` and `event_date` are optional; `event_date` is RFC3339.

Response `201`:

```json
{
  "id": "dh_1",
  "client_id": "usr_1",
  "musician_id": "usr_2",
  "title": "Private acoustic set",
  "location": "Lekki, Lagos",
  "event_date": "2026-08-01T18:00:00Z",
  "price": 300,
  "proposed_by": "usr_1",
  "history": [{ "proposed_by": "usr_1", "price": 300, "created_at": "..." }],
  "status": "pending"
}
```

Status codes: `201`, `400`, `401`, `403`, `405`.

### `GET /direct-hires`

Lists direct hire requests for the authenticated client or musician.

Auth: required.

Query parameters: `status=pending|accepted|declined|cancelled`

Response `200`: array of `DirectHireRequest`.

Status codes: `200`, `401`, `405`, `500`.

### `GET /direct-hires?id=<id>`

Fetches a single direct hire request by ID. Caller must be a participant
(the client or the musician on it).

Status codes: `200`, `403`, `404`.

### `POST /direct-hires/respond`

Accepts or declines the current offer. Caller must be a participant and
must **not** be whoever made the current offer (`proposed_by`). Accepting
creates a contract at the current negotiated price.

Auth: required.

Required body:

```json
{
  "request_id": "dh_1",
  "decision": "accepted"
}
```

`decision` is `"accepted"` or `"declined"`. Response `200`: updated
`DirectHireRequest`.

Status codes: `200`, `400`, `401`, `405`.

### `POST /direct-hires/counter`

Counters the current offer with new terms — updates the price (and any of
title/description/location/event_date that are provided), flips
`proposed_by` to the caller, and appends to `history`. Caller must be a
participant and must not be whoever made the current offer.

Auth: required.

Required body:

```json
{
  "request_id": "dh_1",
  "price": 400,
  "location": "Victoria Island, Lagos"
}
```

Only `request_id` and `price` are required; other fields are optional
partial updates. Response `200`: updated `DirectHireRequest`.

Status codes: `200`, `400`, `401`, `405`.

## Reviews

Reviews attach to a `Contract`, which covers both job-sourced and direct-hire-sourced
contracts. `job_id` appears in the review response only when the contract originated
from a job listing; it's omitted for direct-hire contracts.

### `POST /reviews`

Submits a rating after a contract is completed. Reviewer must be the client or
musician on the contract.

Auth: required.

Required body:

```json
{
  "contract_id": "con_1",
  "rating": 5,
  "comment": "Excellent work"
}
```

Response `201`:

```json
{
  "id": "rev_1",
  "contract_id": "con_1",
  "job_id": "job_1",
  "reviewer_id": "usr_1",
  "reviewee_id": "usr_2",
  "rating": 5,
  "comment": "Excellent work"
}
```

Status codes: `201`, `400`, `401`, `405`.

### `GET /reviews`

Lists reviews received by a user.

Auth: not required by backend.

Query: `user_id=<reviewee_id>`

Response `200`: array of `Review`.

Status codes: `200`, `400`, `405`, `500`.

### `GET /reviews/average`

Returns average rating for a user.

Auth: not required by backend.

Query: `user_id=<reviewee_id>`

Response `200`:

```json
{
  "user_id": "usr_2",
  "average_rating": 5
}
```

Status codes: `200`, `400`, `405`, `500`.

## Notifications

New notifications are pushed in real time over `GET /ws` (see Chat section)
as `{"type": "notification", "data": Notification}` frames, to any recipient
who's currently connected. `GET /notifications` below is for the initial
load only (populating the list on page load / reconnect) — clients should
not poll it.

### `GET /notifications`

Lists notifications for the authenticated user.

Auth: required.

Response `200`: array of `Notification`. When set, `link` is a ready-to-use
frontend path the client should navigate to on click (a booking request →
`/dashboard/talent`, a completed contract → `/contracts/{id}`, a dispute →
`/disputes`, a new chat message → `/messages?with={sender_id}`, etc.).
`contract_id` is a narrower, older mechanism still used by milestone/escrow
notifications to deep-link into that contract's chat thread.

Status codes: `200`, `401`, `405`, `500`.

### `POST /notifications/read`

Marks one notification as read.

Auth: required, owner only.

Required body:

```json
{
  "notification_id": "not_1"
}
```

Response `200`:

```json
{
  "message": "notification marked as read"
}
```

Status codes: `200`, `400`, `401`, `405`, `500`.

## Talent Dashboard

### `GET /talent/dashboard`

Returns musician dashboard data.

Auth: required, role `musician`.

Response `200`:

```json
{
  "musician_id": "usr_2",
  "pending_applications": [],
  "active_jobs": [],
  "completed_jobs": [],
  "contracts": [],
  "average_rating": 5,
  "reviews": [],
  "recommended_jobs": []
}
```

Status codes: `200`, `401`, `403`, `405`, `500`.

## Disputes And Customer Service

### `POST /disputes`

Opens a dispute for a contract. Requester must be a contract participant.

Auth: required.

Required body:

```json
{
  "contract_id": "con_1",
  "reason": "Need admin review"
}
```

Response `201`:

```json
{
  "id": "dsp_1",
  "contract_id": "con_1",
  "client_id": "usr_1",
  "musician_id": "usr_2",
  "opened_by_id": "usr_1",
  "reason": "Need admin review",
  "status": "open"
}
```

Status codes: `201`, `400`, `401`, `405`.

### `GET /disputes`

Lists disputes for the authenticated user.

Auth: required.

Response `200`: array of `Dispute`.

Status codes: `200`, `401`, `405`, `500`.

### `GET /admin/disputes`

Lists all disputes for admin/moderator review.

Auth: required, role `admin` or `moderator`. Moderators have access to this
endpoint and `/admin/disputes/resolve` only — every other `/admin/*`
endpoint (analytics, users, jobs) stays strictly `admin`-only.

Query parameters: `status=open|resolved|closed`

Response `200`: array of `Dispute`.

Status codes: `200`, `401`, `403`, `405`, `500`.

### `POST /admin/disputes/resolve`

Resolves a dispute.

Auth: required, role `admin` or `moderator`.

Auth: required, role `admin`.

Required body:

```json
{
  "dispute_id": "dsp_1",
  "resolution": "Resolved after review"
}
```

Response `200`: updated `Dispute`.

Status codes: `200`, `400`, `401`, `403`, `405`.

## Admin

### `GET /admin/analytics`

Returns platform metrics.

Auth: required, role `admin`.

Response `200`:

```json
{
  "total_users": 3,
  "total_jobs": 1,
  "total_messages": 2,
  "total_contracts": 2,
  "total_disputes": 1
}
```

Status codes: `200`, `401`, `403`, `405`, `500`.

### `GET /admin/users`

Lists all users.

Auth: required, role `admin`.

Response `200`: array of `User`.

Status codes: `200`, `401`, `403`, `405`, `500`.

### `GET /admin/jobs`

Lists all jobs.

Auth: required, role `admin`.

Response `200`: array of `Job`.

Status codes: `200`, `401`, `403`, `405`, `500`.

### `DELETE /admin/jobs`

Deletes a job listing.

Auth: required, role `admin`.

Required body:

```json
{
  "job_id": "job_1"
}
```

Response `200`:

```json
{
  "message": "job deleted successfully by administrator"
}
```

Status codes: `200`, `400`, `401`, `403`, `405`, `500`.

## Media

### `POST /media/upload`

Uploads one or more images, audio, or video files and returns their public
URLs.

Auth: required.

Request: `multipart/form-data`. Repeat the field name `files` once per file
(standard browser behavior when a single `<input type="file" multiple>`
is submitted) — up to 10 files per request, 25MB each. A single field named
`file` also still works for backward compatibility. Accepted content types:
`image/jpeg`, `image/png`, `image/webp`, `image/gif`, `audio/mpeg`,
`audio/wav`, `audio/ogg`, `video/mp4`, `video/webm`, `video/quicktime`.

Response `201`, uploading multiple files:

```json
{
  "files": [
    { "url": "https://api.example.com/uploads/9f2a1c...b3.png", "media_type": "image", "filename": "stage.png" },
    { "url": "https://api.example.com/uploads/71ab04...e2.mp4", "media_type": "video", "filename": "live-set.mp4" }
  ]
}
```

Response `201`, uploading exactly one file — same as above, plus flat
`url`/`media_type` fields at the top level for backward compatibility with
single-file callers:

```json
{
  "url": "https://api.example.com/uploads/9f2a1c...b3.png",
  "media_type": "image",
  "files": [{ "url": "https://api.example.com/uploads/9f2a1c...b3.png", "media_type": "image", "filename": "stage.png" }]
}
```

`media_type` is one of `image`, `audio`, `video` — pass it straight through to
a portfolio item's `media_type` field.

Files are stored on local disk under `MEDIA_UPLOAD_DIR` (default `uploads/`)
and served back from `/uploads/<filename>`. This is intentionally simple for
now; on hosts with ephemeral disks (e.g. Render's free tier) uploaded files
will not survive a redeploy — swap in durable object storage before relying
on this in production.

Status codes: `201`, `400`, `401`, `405`, `413` (file too large), `500`.

### `GET /link-preview`

Unfurls a URL into a title/thumbnail/embeddable player, for showing a rich
preview card when a talent adds an external link (YouTube, Vimeo,
SoundCloud, Spotify, TikTok, or any other site with Open Graph tags) to
their portfolio instead of a bare link.

Auth: required.

Query: `url=<the link to preview>` (must be `http`/`https`; requests to
localhost/private/link-local addresses are rejected).

Response `200`:

```json
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "title": "Rick Astley - Never Gonna Give You Up",
  "thumbnail_url": "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
  "embed_url": "https://www.youtube.com/embed/dQw4w9WgXcQ?feature=oembed",
  "provider": "youtube",
  "media_type": "video"
}
```

`provider` is `youtube`, `vimeo`, `soundcloud`, `spotify`, `tiktok`, or
`link` (the generic Open Graph fallback — `embed_url` is omitted for
`link`, since there's nothing to embed, only a clickable card).
`media_type` is `video`, `audio`, or `link`.

Status codes: `200`, `400`, `401`, `405`, `502` (couldn't fetch/parse the
URL).

## Web Push

Real OS-level push notifications (distinct from the in-app `GET /ws`
realtime feed — this is what fires a notification even when the tab isn't
open). Requires `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` to be set on the
server; if they're not, `/push/subscribe` still works but no push is ever
actually sent (in-app realtime notifications are unaffected either way).

Every notification created anywhere in the app (bookings, milestones,
disputes, reviews, etc.) automatically attempts a Web Push send to all of a
user's registered devices, fire-and-forget, alongside the existing
websocket push.

### `GET /push/vapid-public-key`

Unauthenticated. Returns the VAPID public key the frontend needs to pass as
`applicationServerKey` to `pushManager.subscribe(...)`.

Response `200`: `{"public_key": "..."}`.

### `POST /push/subscribe`

Auth: required. Registers (or updates) a `PushSubscription` for the caller
— call this right after `pushManager.subscribe(...)` resolves in the
browser, with its `.toJSON()` shape.

Required body:

```json
{
  "endpoint": "https://fcm.googleapis.com/fcm/send/...",
  "keys": { "p256dh": "...", "auth": "..." }
}
```

Response `201`: the saved subscription. Status codes: `201`, `400`, `401`, `405`.

### `POST /push/unsubscribe`

Auth: required. Removes one of the caller's subscriptions by endpoint.

Required body: `{"endpoint": "..."}`. Response `200`. Status codes: `200`, `400`, `401`, `405`.

## Wallet

Authenticated. The wallet acted on is always the caller's own (derived from
the JWT) — there is no `user_id` parameter. Not backed by a real payment
processor yet (no external money movement), but state is real, persisted in
MongoDB, and shared correctly between all users — this is what Milestones
below builds on.

### `GET /wallet`

Returns (and lazily creates, on first call) the caller's wallet.

Response `200`:

```json
{
  "user_id": "usr_1",
  "balance": 400,
  "escrow_balance": 100,
  "total_earned": 0,
  "total_spent": 100,
  "updated_at": "2026-07-02T10:00:00Z"
}
```

Status codes: `200`, `401`, `405`, `500`.

### `POST /wallet/deposit`

Required body: `{"amount": 100}`. Response `200`: the updated wallet.

### `POST /wallet/withdraw`

Required body: `{"amount": 100}`. Fails with `400` if `amount` exceeds the
current balance. Response `200`: the updated wallet.

### `GET /wallet/transactions`

Returns the caller's transaction ledger, newest first. Each entry has
`type` (`deposit`, `withdrawal`, `escrow_hold`, `escrow_release`,
`payment_received`), `amount`, `description`, `created_at`.

## Milestones

Authenticated. A milestone belongs to a `Contract` and moves through:

```
proposed → accepted → funded → released
   ↳ (counter keeps status "proposed", flips who's offering)
              ↳ rejected
```

Either party on the contract can propose a milestone; the **other** party
must accept, reject, or counter it before it can be funded. A counter-offer
updates the title/amount/due date, flips `proposed_by` to the counterer, and
appends to `history` — mirroring how direct-hire booking negotiation works
(see Contracts And Direct Hire). Only the **client** can
fund an accepted milestone (moves money from their wallet balance into
their own escrow) or release a funded one (moves money from their escrow
into the musician's wallet balance). Every transition creates a real,
persisted `Notification` for the other party — which also pushes instantly
over `GET /ws` (see Chat section) — and the notification includes
`contract_id` so the frontend can deep-link back to that contract's chat
thread.

### `GET /milestones?contract_id=<id>`

Lists milestones for a contract. Caller must be a participant (client or
musician) on it.

### `POST /milestones`

Propose one or more milestones.

Required body:

```json
{
  "contract_id": "ctr_1",
  "milestones": [
    { "title": "Rehearsal complete", "amount": 100, "due_date": "2026-08-01T00:00:00Z" }
  ]
}
```

`due_date` is optional. Response `201`: the created milestones (`status: "proposed"`).

### `POST /milestones/accept`

Required body: `{"contract_id": "ctr_1", "milestone_id": "ms_1"}`. Caller
must be the participant who did **not** propose it. Fails if the milestone
isn't `proposed`.

### `POST /milestones/reject`

Same body/authorization as accept. Sets status to `rejected` (terminal).

### `POST /milestones/counter`

Counters the current offer with new terms. Caller must be a participant and
must **not** be whoever made the current offer (`proposed_by`). Fails if the
milestone isn't `proposed`.

Required body:

```json
{
  "contract_id": "ctr_1",
  "milestone_id": "ms_1",
  "amount": 150,
  "title": "Rehearsal complete (revised)",
  "due_date": "2026-08-05T00:00:00Z"
}
```

`title` and `due_date` are optional partial updates; `amount` is required.
Response `200`: updated milestone, with `proposed_by` flipped to the caller
and a new entry appended to `history`.

### `POST /milestones/fund`

Same body shape. Caller must be the contract's client. Milestone must be
`accepted`. Fails with `400` if the client's wallet balance is insufficient.
Moves `amount` from the client's `balance` to their `escrow_balance`.

### `POST /milestones/release`

Same body shape. Caller must be the contract's client. Milestone must be
`funded`. Moves `amount` out of the client's `escrow_balance` and into the
musician's `balance` (and `total_earned`).

Status codes across all milestone endpoints: `200`/`201`, `400`
(validation/authorization/state errors, e.g. wrong party, wrong status,
insufficient balance), `401`, `405`.

## Simulation Test

The dummy client/musician/admin flow is covered by:

```bash
go test ./internal/delivery/http -run TestSimulateClientMusicianAPIFlow -v
```

The test creates:

| Role | Email |
| --- | --- |
| Client | `client@example.com` |
| Musician | `musician@example.com` |
| Admin | `admin@example.com` |

It simulates signup, login, password reset, profiles, musician search, job posting, application, REST chat, WebSocket chat, application acceptance, contract creation/completion, direct hire, reviews, notifications, dashboard, disputes, admin endpoints, wallet, and milestone escrow (propose/accept/fund/release).

## Postman Documentation

Postman does not publish this Markdown file directly as runnable API documentation. The normal flow is to import a Postman Collection or API specification, then let Postman generate documentation from that collection.

Import these files:

```text
docs/postman/Gigpurse.postman_collection.json
docs/postman/Gigpurse.postman_environment.json
```

Recommended Postman workflow:

1. Open Postman.
2. Click **Import**.
3. Import both JSON files above.
4. Select the **Gigpurse Local** environment.
5. Run the signup/login requests first so Postman stores `client_token`, `musician_token`, `admin_token`, and IDs.
6. Open the collection overview and choose **View complete documentation**.
7. Use Postman's publish/share documentation option when you are ready to expose it to other developers.

For local requests, run the backend first:

```bash
go run cmd/gigpurse/main.go
```

Set `ALLOW_ADMIN_SIGNUP=true` only in a local/dev environment if you want the admin signup request to work from Postman.
