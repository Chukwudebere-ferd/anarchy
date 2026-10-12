# Anarchy

Anarchy is an early-user simulation product. We train bots with realistic mock data so startup owners can authorize them on their own platforms and kickstart activity: signups, posts, comments, and other tasks depending on the platform. Bots run as scheduled workers (cron-like), not chatbots.

Example: a startup needs 500 signups. They spin up 300 trained bots while real users who sign up complete the remaining 200 to hit the goal.

**Live API:** `https://anarchy.zeabur.app` — open, no keys.

## Quickstart for agents (any framework)

Give your agent this repo as a skill/reference, or call directly:

```sh
# Get trained bots (max 50 per request)
curl -X POST https://anarchy.zeabur.app/v1/bots \
  -H "Content-Type: application/json" \
  -d '{"count": 25, "schedule": "@every 2h", "jitter_seconds": 600}'
```

Response: N ready-to-run bots, each with identity (name, age, profile — no contact details), full character (traits, backstory, quirks, routine, goals, knowledge, examples), schedule, and permissions.

Then, on the platform owner's staging environment:

1. **Schedule** each bot: one cron row per bot, `schedule` ± `jitter_seconds`.
2. **Execute** tasks via the owner's tools (MCP server, API clients, SDK calls). Task types: `signup`, `post`, `comment`, `view`, `follow`, job-board pack (`job_search`, `save_job`, `apply`, `post_job`), and any custom type — unknown types pass through; payloads are free-form JSON objects.
3. **Enforce holds**: task types listed in `permissions.need_approval` (financial, destructive, privacy-sensitive, irreversible, external-publish, real-user-contact) must be staged and reported — never executed — until the platform owner approves.
4. **Rate limit**: 10 actions/min per bot. Log every outcome.

Other endpoints:

- `GET /v1/eval` — training quality report (per-character pass rates)
- `GET /healthz` — liveness

Limits: 50 bots/request, 10 requests/hour per IP. Contact details (email/phone) are intentionally excluded from all responses.

See [docs/api.md](docs/api.md) for full details and [docs/onboarding.md](docs/onboarding.md) to run a pilot.

## Goals

- Create and manage bots in bulk.
- Give each bot a structured identity and realistic mock data (name, age, email, phone, profile, description).
- Train behavior using explicit traits, backstory, quirks, routine, goals, knowledge, and task examples — every bot gets its own character line.
- Run bots as scheduled workers through the owner's agent with least-privilege access.
- Hold sensitive actions for client approval.
- Evaluate whether trained bots complete tasks realistically and consistently.
- Export approved configurations for any agent platform.

## Core Concepts

### 1. Bot Identity

A bot identity contains its user-facing and operational data:

- Name
- Age
- Email (stored encrypted, never exposed via API)
- Phone number (stored encrypted, never exposed via API)
- Profile
- Description

Identities and characters are separate, so any identity can pair with any character — or one-to-one, as bulk training does now.

Contact details are sensitive data: encrypted at rest (AES-256-GCM), access-restricted, deletable, with a clear basis for use.

### 2. Behavioral Training

Training uses structured data instead of a single unstructured prompt. Each bot's character defines:

- Personality traits
- Communication style
- Backstory and daily routine
- Quirks (how this specific human writes and acts)
- Goals and priorities
- Relevant knowledge
- Example tasks and results (task in, done/held result out)

The objective is realistic, consistent simulated activity. The platform must not be used for deceptive impersonation of real people or to bypass another platform's identity and access controls. Live simulation runs only with the platform owner's authorization.

### 3. Behavior Evaluation

Training is not complete until the bot is evaluated. Scenarios test:

- Task completion on schedule
- Realism and consistency of mock data and actions
- Boundary enforcement (holds for approval where required)
- Appropriate requests for client approval
- Quality of action logs

Every character version has an eval history — regressions are caught before export.

### 4. Agent Integration

Bot configurations work with any agent framework. The core export is framework-neutral JSON; adapters render it per platform:

- **Generic bundle** (`--format generic`): plain system prompts (`agents/*.md`), `roster.json`, `rules.md` — CrewAI, LangChain, AutoGen, custom runners
- **OpenCode bundle** (`--format opencode`): `opencode.json`, `.opencode/agents/*.md`, `anarchy-bots.json`, rules — spec-verified against OpenCode v2 docs

Personality and tool permissions stay separate so connecting a bot never grants unrestricted access.

## API Reference

Full reference: [docs/api.md](docs/api.md).

| Endpoint | Method | Purpose |
|---|---|---|
| `/v1/bots` | POST | Issue N trained bots (`count`, `schedule`, `jitter_seconds`, `seed`) |
| `/v1/eval` | GET | Training quality report |
| `/healthz` | GET | Liveness |

## Safety Principles

- Treat email addresses and phone numbers as sensitive data — encrypted, never exposed.
- Run live simulation only with the platform owner's authorization, on staging first.
- Do not allow bots to impersonate real people; disclose bot use where end users are affected.
- Validate training material and configuration inputs to reduce prompt-injection and persona-manipulation risks.
- Require client approval for destructive, financial, privacy-sensitive, irreversible, external-publish, or real-user-contact actions.
- Give every integration only the permissions it needs.
- Log agent requests, tool calls, approvals, and outcomes without exposing secrets.
- Provide a clear way to pause, disable, or delete a bot.

## Self-Hosting (operators)

The hosted API is for testing. To run your own instance:

```sh
git clone <repo> && cd anarchy
cp .env.example .env          # set DATABASE_URL + ANARCHY_CONTACT_KEY
docker compose up -d db       # or point DATABASE_URL at your managed Postgres
go build -o anarchy-server.exe ./cmd/server
go build -o anarchy-api.exe   ./cmd/api
./anarchy-server.exe --migrate
./anarchy-seed.exe -n 100 -seed 7 -domain yourdomain.com -out functions/identities.bulk.json
./anarchy-server.exe --import-to-db --use-db -n 50
./anarchy-api.exe             # serves :8080 — same endpoints as the hosted API
```

Environment:

- `DATABASE_URL` — Postgres connection string
- `ANARCHY_CONTACT_KEY` — 32-byte key, base64: `openssl rand -base64 32` (never commit; losing it makes encrypted contacts unreadable)
- `PORT` — API port (default 8080)

## Technical Direction

Go + Postgres.

```text
anarchy/
├── cmd/
│   ├── api/                 # Open HTTP API (public, keyless)
│   ├── server/              # Training/export CLI (migrate, import, randomize, export, eval, bundle)
│   └── seed/                # Bulk identity/character JSON generator (local or randomuser.me)
├── internal/
│   ├── api/                 # HTTP handlers, rate limiting, public bot shape
│   ├── identity/            # Bot identities, encrypted contact store
│   ├── character/           # Behavior configuration and versioning (NPC-style)
│   ├── training/            # Identity x character randomization + schedules
│   ├── evaluation/          # Scenarios, scoring, results
│   ├── integrations/        # Agent bundles (generic + OpenCode), permissions
│   ├── execution/           # Task contract, schedules, approval gate
│   └── platform/            # DB pool, AES-256-GCM contact encryption
├── migrations/              # 0001 init, 0002 age, 0003 schedules/tasks, 0004 npc
├── functions/               # Identity/character JSON, bulk data, exports
├── docs/                    # api.md, onboarding.md
├── Dockerfile               # Builds cmd/api (Zeabur/any container host)
├── docker-compose.yml       # Local Postgres
├── go.mod
└── README.md
```

## Project Status

Live. Hosted open API serving trained bots, 553 identities + 553 unique character lines in Postgres (contacts encrypted), 1655/1650+ eval examples passing, agent bundles for any framework, approval-gated task contract, first pilot completed end-to-end on a live blog platform (isolated test DB, kill-switch verified, prod untouched). Next: pilots on additional SaaS platforms and continued training depth.
