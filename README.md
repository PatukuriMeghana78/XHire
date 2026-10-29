# XHire

A two-sided job-application platform: candidates track applications, upload a résumé, and see *why* a job is or is not a match; recruiters post jobs and get candidates ranked with an explanation for each score. XHire also imports a candidate's own Gmail replies and turns them into application-status updates through a Python NLP service.

The repository contains three independently runnable services plus their client code:

| Service | Path | Runtime | Default port |
| --- | --- | --- | --- |
| Web client | `Frontend/` | React 19 + TanStack Start (Vite) | 5173 |
| API server | `backend/` | Go 1.25 + Gin + GORM | 8080 |
| NLP service | `nlp/` | Python 3 + FastAPI + uvicorn | 8000 |
| Database | — | PostgreSQL | 5432 (db `job_tracker`) |

## Problem statement

Job seekers lose track of which company replied, which application went quiet, and what a rejection actually meant. Existing trackers require manual updates, and generic "match score" features give a number without a reason. XHire addresses this by:

1. Watching the candidate's **own mailbox** for recruiter replies and converting them into status suggestions instead of relying on manual data entry.
2. Producing **explainable** match and ranking output — every score is accompanied by weighted factors, matched skills and missing skills, computed server-side.
3. Giving recruiters the same explanation from the other side: an ordered candidate list per job with the reasons behind each position.

## Implemented features

| Area | Status | Notes |
| --- | --- | --- |
| Candidate/recruiter registration and login | Implemented | bcrypt (cost 12), HS256 JWT valid 24 h, role-scoped route groups |
| Job posting, editing, deletion | Implemented | Recruiter-owned jobs; candidate-facing list and detail |
| Apply to a job | Implemented | Creates an application and a recruiter-visible candidate record |
| Résumé upload and download | Implemented | 5 MiB cap, `.pdf`/`.docx`/`.txt`, stored outside the web root, downloads authorized per owner |
| Skill gaps and job match | Implemented | Backend-computed factors; no inference in the browser |
| Recruiter ranking + explanation | Implemented | Ordered candidates per job with per-candidate reasons |
| Gmail connect, sync, status, disconnect | Implemented | Offline consent grant, per-user token store, 3-minute background poller |
| Email → status classification | Implemented | FastAPI `/classify`; heuristic fallback is labelled as such in the UI |
| Notifications | Implemented | Created on email detection and status change |
| Analytics | Implemented | Dashboard stats, status distribution, weekly summary, real time-to-response |
| Live Gmail round trip | **Not verified** | All Gmail flows were exercised against a scratch database with a deliberately invalid OAuth client; no real Google account was authorized during development |
| Payment, messaging, mobile apps | Not implemented | Out of scope |

## Architecture

```
Browser (React 19 / TanStack Start, :5173)
        │  REST + Bearer JWT  (VITE_API_BASE_URL)
        ▼
Go + Gin API (:8080)  ──GORM──►  PostgreSQL (job_tracker)
        │        │
        │        └── HTTP ──►  FastAPI NLP service (:8000)
        │                      /classify /analyze /health-score /skill-match
        └── OAuth2 + Gmail API (gmail.modify) ⇄ Google
              background poller every 3 minutes: Q("is:unread newer_than:2d")
```

All AI/explainability output is produced by the Go backend or the NLP service. The frontend only transports it.

### Repository layout

```
.
├── Frontend/            # TanStack Start app (React 19, Tailwind 4, shadcn/ui-style components)
│   ├── src/routes/      # file-based routes: candidate.*, recruiter.*, login, register
│   ├── src/services/    # one module per API domain (auth, jobs, applications, email, ai, …)
│   ├── src/components/  # ui/, ai/, auth/, common/, email/, jobs/, layout/
│   └── src/lib/api/     # fetch client, token store, ApiError
├── backend/             # Go API
│   ├── main.go          # wiring, AutoMigrate, Gmail poller goroutine
│   └── internal/
│       ├── config/      # env loading + JWT secret validation
│       ├── database/    # Postgres connection, mock helpers
│       ├── models/      # GORM models
│       ├── dto/         # request/response shapes (no credentials leave here)
│       ├── repositories/
│       ├── services/    # auth, jobs, applications, gmail, nlp client, match, analytics, notifications
│       ├── handlers/
│       └── routes/      # /api/* legacy and /api/v1/* XHire surface
└── nlp/                 # FastAPI service: parsing, preprocessing, role/company extraction,
    └── src/             # status classification, confidence scoring, health scoring
```

## Prerequisites

- Go 1.25 or newer
- Node.js 20+ and npm
- Python 3.10 – 3.13 with `venv`
- PostgreSQL 14+ running locally, reachable as a superuser role you can log in with

## Environment configuration

Create the files below yourself — neither is tracked by Git, and the repository intentionally ships no `.env.example`.

`backend/.env`

| Variable | Purpose | Example |
| --- | --- | --- |
| `DB_URL` | PostgreSQL connection string | `host=localhost user=postgres password=<your-password> dbname=job_tracker port=5432 sslmode=disable` |
| `PORT` | API listen port | `8080` |
| `JWT_SECRET` | HS256 signing key; **required**, minimum 32 bytes, refuses known weak defaults | `<32+ random characters>` |
| `GOOGLE_CLIENT_ID` | OAuth web-application client id | `<google-client-id>.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | OAuth client secret | `<google-client-secret>` |
| `GOOGLE_REDIRECT_URL` | Must match a redirect URI registered in Google Cloud Console | `http://localhost:8080/api/email/oauth/callback` |
| `NLP_SERVICE_URL` | Base URL of the FastAPI service | `http://localhost:8000` |
| `FRONTEND_URL` | Optional. Origin the Gmail callback redirects back to; defaults to `http://localhost:5173` | `http://localhost:5173` |

`Frontend/.env`

| Variable | Purpose |
| --- | --- |
| `VITE_API_BASE_URL` | Base URL of the Go API, e.g. `http://localhost:8080`. When unset the client falls back to bundled mock fixtures — Gmail deliberately has **no** mock data, so the card reports that a backend is required instead of pretending to be connected. |

## Database setup

XHire uses GORM `AutoMigrate` on startup — there are no `.sql` files to apply.

1. Create the database yourself (the app never creates it):
   ```
   createdb job_tracker
   ```
2. Point `DB_URL` at it and start the backend. `main.go` migrates this model set:
   `User`, `Application`, `StatusHistory`, `EmailEvent`, `SuggestedUpdate`, `GmailToken`, `Notification`, `Job`, `CandidateProfile`, `RecruiterProfile`, `Resume`.

AutoMigrate is additive: it creates missing tables and columns and does not drop or narrow existing ones. New columns such as `gmail_tokens.needs_reconnect` and `reconnect_reason` appear on the next boot without manual work. There is deliberately no unique index on `gmail_tokens.user_id` because duplicate rows in existing databases were never audited; the application resolves duplicates by selecting the newest row per user.

## Running the project

```bash
# 1. NLP service
cd nlp
python -m venv venv && . venv/Scripts/activate      # Windows Git Bash
pip install -r requirements.txt
python -m uvicorn src.nlp_api:app --host 127.0.0.1 --port 8000

# 2. Go backend (reads backend/.env via godotenv)
cd ../backend
go mod download
go run main.go                                      # :8080

# 3. Web client
cd ../Frontend
npm install
npm run dev                                         # :5173
```

Sign up at `http://localhost:5173/register` as a candidate or a recruiter. The backend exits at startup if `JWT_SECRET` is missing, shorter than 32 characters, or set to a known insecure default.

## API surface

`/api/v1` is the surface the React client uses. Everything except `POST /auth/login` and `POST /auth/register` requires `Authorization: Bearer <jwt>`; the `candidate` and `recruiter` groups additionally enforce the role claim, returning `403 insufficient role`.

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/v1/auth/register`, `/auth/login` | Create account, obtain token |
| GET/POST | `/api/v1/auth/me`, `/auth/logout` | Current user, logout |
| GET | `/api/v1/jobs`, `/jobs/:id` | Browse and inspect postings |
| POST | `/api/v1/jobs/:id/apply` | Apply |
| GET/PUT | `/api/v1/candidate/profile` | Own profile |
| GET/POST | `/api/v1/candidate/resume` | Résumé metadata, upload |
| GET | `/api/v1/resumes/:resumeId/download` | Authorized file download |
| GET | `/api/v1/resumes/:resumeId/analysis` | Stored résumé analysis |
| GET | `/api/v1/candidate/applications`, `/skill-gaps`, `/jobs/:jobId/match` | Own pipeline and explainable match |
| GET | `/api/v1/candidate/email/status` | Gmail state: connected / needs-reconnect / reason |
| GET | `/api/v1/candidate/email/messages` | Imported recruitment emails for the caller only |
| GET | `/api/v1/candidate/email/connect-url` | CSRF-protected Google authorization URL |
| POST | `/api/v1/candidate/email/sync`, `/disconnect` | Poll now, revoke local grant |
| POST / PUT / DELETE | `/api/v1/jobs`, `/jobs/:id` | Create, edit, close a posting (recruiter-only) |
| GET | `/api/v1/recruiter/jobs`, `/recruiter/applications`, `/recruiter/jobs/:jobId/applications` | Recruiter pipeline |
| PATCH | `/api/v1/recruiter/applications/:id/status` | Move a candidate |
| GET | `/api/v1/candidates/:id`, `/candidates/:id/resume` | Inspect a candidate |
| GET | `/api/v1/recruiter/jobs/:jobId/ranking`, `/recruiter/jobs/:jobId/candidates/:candidateId/explanation` | Ranked candidates and reasons |

The older `/api/*` surface (`/api/applications`, `/api/analytics/*`, `/api/notifications`, `/api/suggestions`, `/api/email/*`, `/api/settings`, `/api/auth/signup`, `/api/email/oauth/callback`) remains available for the legacy Applylitics client and for the Gmail OAuth callback.

## Gmail integration

### Google Cloud Console configuration

1. Create a project, enable the **Gmail API**.
2. Configure the OAuth consent screen. While it is in **Testing**, only explicitly listed test users may authorize; publish it to **In production** for general use.
3. Create an **OAuth client ID** of type **Web application** and add the authorized redirect URI exactly as `GOOGLE_REDIRECT_URL` (default `http://localhost:8080/api/email/oauth/callback`).
4. Put the client id and secret in `backend/.env`. Nothing is hardcoded and no credential file is committed.

### How the flow works

- The client asks `GET /api/v1/candidate/email/connect-url`. The server issues a 256-bit single-use state token bound to the caller, valid for 10 minutes, and returns an authorization URL carrying `access_type=offline` **and** `prompt=consent` with the `gmail.modify` scope. `prompt=consent` is required: without a fresh consent screen Google does not return a refresh token, and the grant silently dies after one hour.
- `GET /api/email/oauth/callback` validates the state (rejecting replays and expired values), exchanges the code, and stores the grant for that user. Google's own `?error=` responses are converted into a human-readable reason and redirected back to the dashboard.
- Storage upserts per user and **never overwrites a stored refresh token with an empty value**; refreshed tokens are written back, and a rotated response that omits the refresh token keeps the existing one.
- A background poller runs every 3 minutes per user with a per-user lock, so a manual "Sync now" cannot overlap it. Only unread messages from the last two days are read, and only their classification results are persisted — no message bodies are stored.
- Failures are classified: a dead or revoked grant (`invalid_grant`, or a grant with no refresh token) is persisted as `needs_reconnect`, the poller stops retrying it, and the dashboard offers **Reconnect Gmail**. A rejected OAuth client (`invalid_client`, `accessNotConfigured`, `invalid_scope`) returns `503 gmail_misconfigured` and leaves the user's grant untouched, because reauthorizing cannot fix a server-side configuration error. Transient failures remain retryable.

### Known limitations

- The live round trip with a real Google account has not been validated from this workspace; see the testing section.
- `gmail.modify` is requested rather than `gmail.readonly`, so the poller can mark processed messages as read.
- Detection quality depends on the NLP service; when it cannot classify confidently, the result is labelled `heuristic` in the UI rather than presented as certain.
- The poller runs inside the API process. Multiple API instances would each poll, so scale by running one instance or moving the poller to a dedicated worker.

## Testing

```bash
cd backend
gofmt -l ./internal ./main.go   # only files you edited; the tree is CRLF on Windows
go vet ./...
go test ./...

cd ../Frontend
./node_modules/.bin/tsc --noEmit   # not `npx tsc`, which resolves to an unrelated package
npm run build
npm run lint
```

Results from the most recent run on this working tree (2026-10-29):

- `go build ./...`, `go vet ./...` and `go test ./...` passed. The suite covers OAuth state issue/validate/expiry/replay, refresh-token write-back and preservation, dead-grant classification, poller skip behaviour, per-user locks, error-to-HTTP mapping, and the assertion that stored credentials never serialize to JSON.
- `tsc --noEmit` exited 0 and `npm run build` completed successfully.
- HTTP contract checks against a throwaway PostgreSQL and backend on scratch ports: 28/28 and 18/18 assertions passed, including per-user message isolation, `401` without a token, `403` for the wrong role, `409 gmail_not_connected`, `409 gmail_reconnect_required` without contacting Google, `503 gmail_misconfigured`, and no token material in any response body.
- Browser pass over the candidate dashboard confirmed the Gmail card's loading, not-connected, connected-with-results, connected-empty and needs-reconnect states, and that **Connect Gmail** reaches Google's live authorization endpoint.

Not covered: a successful authorization and import against a real mailbox, NLP unit tests (the `nlp/` service has none), and any load or concurrency testing.

## Security considerations

- Secrets live only in untracked `.env` files; `.gitignore` excludes `.env*`, client-secret and credential JSON, key material, virtual environments, `node_modules`, build output and runtime uploads.
- Passwords use bcrypt at cost 12; JWT validation pins the HMAC signing method, so an `alg: none` or attacker-chosen algorithm is rejected. `JWT_SECRET` is validated at startup.
- Gmail access tokens and refresh tokens carry `json:"-"` and are never returned by any endpoint; status responses expose only booleans and a human-readable reason.
- OAuth error text can embed raw token-endpoint responses, so failures are logged as structured summaries with credential-shaped fields redacted.
- The Gmail callback state parameter is random, user-bound, single-use and time-limited, which blocks CSRF and code-injection replay against the callback.
- Every candidate and recruiter query is scoped by the authenticated user id taken from the token; no endpoint accepts a target user id from the client for private data.
- Résumé files are stored outside the served root with sanitized names, an extension allow-list and a 5 MiB limit, and are streamed only through an ownership check.
- CORS is limited to the local client origins with credentials enabled.
- Known gaps: no rate limiting, no token revocation list (a logout only discards the client-side token until the 24 h expiry), no request-body size ceiling outside the upload handler, and no automated security scanning in CI.

## Future enhancements

- Move Gmail polling to a dedicated worker with a queue and backoff, so multiple API instances are safe.
- Persist OAuth refresh rotation with an at-rest encryption layer for stored tokens.
- Add rate limiting, short-lived access tokens with refresh, and a server-side revocation list.
- Broaden test coverage to the NLP service and to end-to-end Gmail flows with a real test account.
- Surface per-application email threads and let a candidate confirm or reject a suggested status inline.
- Introduce recruiter-side inbox parsing and team workspaces behind an organization model.
