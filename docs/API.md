# Gigpurse Backend API Documentation

Base URL for local development: `http://localhost:8080`

Authentication uses JWT bearer tokens:

```http
Authorization: Bearer <token>
```

Standard error response uses plain text from `http.Error`, for example:

```text
unauthorized: only musicians can apply for jobs
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

Wallet/payment/escrow functionality is still pending product-wise. Current wallet endpoints are prototype-only and in-memory.

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
  "name": "Demo Client"
}
```

Roles: `client`, `musician`. `admin` is allowed only when `ALLOW_ADMIN_SIGNUP=true`.

Response `201`:

```json
{
  "id": "usr_1",
  "email": "client@example.com",
  "role": "client",
  "name": "Demo Client",
  "bio": "",
  "location": "",
  "client_profile": {
    "company_name": ""
  },
  "created_at": "2026-06-19T18:58:33Z",
  "updated_at": "2026-06-19T18:58:33Z"
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
  "token": "<jwt>",
  "user": {
    "id": "usr_1",
    "email": "client@example.com",
    "role": "client",
    "name": "Demo Client"
  }
}
```

Status codes: `200`, `400`, `401`, `405`.

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
    "instrument": "Guitar",
    "genre": "Afrobeats",
    "experience_years": 7,
    "portfolio": [
      {
        "title": "Live Session",
        "description": "Recorded guitar session",
        "url": "https://example.com/session.mp4"
      }
    ]
  }
}
```

Response `200`: updated `User`.

Status codes: `200`, `400`, `401`, `405`, `500`.

### `GET /musicians`

Searches musician profiles.

Auth: not required by backend.

Query parameters:

| Name | Description |
| --- | --- |
| `genre` | Case-insensitive genre filter. |
| `instrument` | Case-insensitive instrument filter. |
| `location` | Case-insensitive location filter. |
| `min_exp` | Minimum years of experience. |
| `sort_by` | `experience`, `newest`, or `rating` placeholder. |
| `sort_order` | `asc` or `desc`. |

Example:

```http
GET /musicians?genre=Afrobeats&instrument=Guitar&location=Lagos&min_exp=3&sort_by=experience
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
      "instrument": "Guitar",
      "genre": "Afrobeats",
      "experience_years": 7
    }
  }
]
```

Status codes: `200`, `405`, `500`.

## Jobs And Applications

### `POST /jobs`

Creates a job listing.

Auth: required, role `client`.

Required body:

```json
{
  "title": "Afrobeats guitar session",
  "description": "Need guitar for a studio session",
  "instrument": "Guitar",
  "genre": "Afrobeats",
  "location": "Lagos",
  "budget": 500
}
```

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
  "status": "open"
}
```

Status codes: `201`, `400`, `401`, `403`, `405`.

### `GET /jobs`

Lists jobs or fetches one job by ID.

Auth: not required by backend.

Query parameters:

| Name | Description |
| --- | --- |
| `id` | If present, returns one job. |
| `status` | `open`, `active`, `completed`, `disputed`. |
| `genre` | Genre filter. |
| `instrument` | Instrument filter. |
| `location` | Location filter. |
| `min_budget` | Minimum budget. |
| `max_budget` | Maximum budget. |
| `sort_by` | `newest`, `budget`, `applications`, `popularity`, `relevance`. |
| `sort_order` | `asc` or `desc`. |
| `max_applications` | Only jobs with application count less than or equal to this value. |

Response `200`:

```json
[
  {
    "id": "job_1",
    "status": "open",
    "title": "Afrobeats guitar session",
    "budget": 500
  }
]
```

Status codes: `200`, `404`, `405`, `500`.

### `GET /jobs/recommended`

Returns personalized open gig recommendations for a musician.

Auth: required, role `musician`.

Query parameters: `limit` defaults to `10`, max practical limit is `20`.

Response `200`: array of `Job`.

Status codes: `200`, `400`, `401`, `403`, `405`.

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

Sends a chat message. External payment/contact terms are censored.

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

### `GET /chats/ws?token=<jwt>`

WebSocket endpoint for realtime chat.

Auth: JWT in `token` query parameter or `Authorization` header.

Client message:

```json
{
  "recv_id": "usr_2",
  "content": "Realtime hello"
}
```

Server sends `ChatMessage` JSON to sender and online receiver.

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

### `POST /direct-hires`

Creates a direct hire request from a client to a musician.

Auth: required, role `client`.

Required body:

```json
{
  "musician_id": "usr_2",
  "title": "Private acoustic set",
  "description": "Direct hire for a private event",
  "price": 300
}
```

Response `201`:

```json
{
  "id": "dh_1",
  "client_id": "usr_1",
  "musician_id": "usr_2",
  "title": "Private acoustic set",
  "price": 300,
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

### `POST /direct-hires/respond`

Allows a musician to accept or decline a direct hire request. Accepting creates a contract.

Auth: required, role `musician`.

Required body:

```json
{
  "request_id": "dh_1",
  "decision": "accepted"
}
```

Response `200`: updated `DirectHireRequest`.

Status codes: `200`, `400`, `401`, `403`, `405`.

## Reviews

### `POST /reviews`

Submits a rating after a job is completed. Reviewer must be the client or hired musician.

Auth: required.

Required body:

```json
{
  "job_id": "job_1",
  "rating": 5,
  "comment": "Excellent work"
}
```

Response `201`:

```json
{
  "id": "rev_1",
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

### `GET /notifications`

Lists notifications for the authenticated user.

Auth: required.

Response `200`: array of `Notification`.

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

Lists all disputes for admin review.

Auth: required, role `admin`.

Query parameters: `status=open|resolved|closed`

Response `200`: array of `Dispute`.

Status codes: `200`, `401`, `403`, `405`, `500`.

### `POST /admin/disputes/resolve`

Resolves a dispute.

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

## Prototype Wallet

These endpoints exist but are pending for real product use. They are unauthenticated and backed by in-memory storage.

### `POST /wallet`

Creates an in-memory wallet.

Required body:

```json
{
  "user_id": "usr_1"
}
```

Response `201`:

```json
{
  "message": "wallet created successfully"
}
```

Status codes: `201`, `400`, `405`, `500`.

### `GET /wallet`

Gets an in-memory wallet balance.

Query: `user_id=<user_id>`

Response `200`:

```json
{
  "user_id": "usr_1",
  "balance": 100
}
```

Status codes: `200`, `400`, `404`, `405`.

### `POST /wallet/deposit`

Adds money to an in-memory wallet.

Required body:

```json
{
  "user_id": "usr_1",
  "amount": 100
}
```

Response `200`:

```json
{
  "message": "deposit successful"
}
```

Status codes: `200`, `400`, `405`, `500`.

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

It simulates signup, login, password reset, profiles, musician search, job posting, application, REST chat, WebSocket chat, application acceptance, contract creation/completion, direct hire, reviews, notifications, dashboard, disputes, admin endpoints, and prototype wallet endpoints.

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
