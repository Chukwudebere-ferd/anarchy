# Anarchy

Anarchy is an early-user simulation product. We train bots with realistic mock data so startup owners can authorize them on their own platforms and kickstart activity: signups, posts, comments, and other tasks depending on the platform. Bots run as scheduled workers (cron-like), not chatbots.

Example: a startup needs 500 signups. They spin up 300 trained bots while real users who sign up complete the remaining 200 to hit the goal.

## Goals

- Create and manage bots in bulk.
- Give each bot a structured identity and realistic mock data (name, age, email, phone, profile, description).
- Train behavior using explicit traits, goals, knowledge, boundaries, and task examples.
- Run bots as scheduled workers through the owner's agent with least-privilege access.
- Hold sensitive actions (financial, destructive, privacy-sensitive, irreversible, external-publish, real-user-contact) for client approval.
- Evaluate whether trained bots complete tasks realistically and consistently.
- Export approved configurations for compatible agent platforms with an audit trail.

## Core Concepts

### 1. Bot Identity

A bot identity contains its user-facing and operational data, including:

- Name
- Age
- Email
- Phone number
- Profile
- Description

Identities and characters are separate, so any identity can pair with any character (e.g. identity "Ferd" with one character, identity "Sophia" with the "ferd" character).

Contact details are sensitive data. The platform must encrypt them, restrict access, support deletion, and require a clear basis for their use.

### 2. Behavioral Training

Training uses structured data instead of a single unstructured prompt. A bot's behavior can define:

- Personality traits
- Action patterns and schedule discipline
- Goals and priorities
- Relevant knowledge
- Behavioral boundaries
- Example tasks and results (task in, done/held result out)

The objective is realistic, consistent simulated activity. The platform must not be used for deceptive impersonation of real people or to bypass another platform's identity and access controls. Live simulation runs only with the platform owner's authorization.

### 3. Behavior Evaluation

Training is not complete until the bot can be evaluated. Evaluation scenarios can test:

- Task completion on schedule
- Realism and consistency of mock data and actions
- Boundary enforcement (holds for approval where required)
- Appropriate requests for client approval
- Quality of action logs

Each bot should have a versioned training configuration and a history of evaluation results so regressions can be identified.

### 4. Agent Integration

Bot configurations are adapted for the owner's agent (such as OpenCode). Task instructions and tool permissions remain separate so connecting a bot does not automatically grant unrestricted access.

Agent integrations support:

- Explicit bot configuration export
- Versioned behavior data
- Least-privilege tool permissions (scheduled mock-data tasks run freely; sensitive types hold for approval)
- Client approval for sensitive actions
- Rate limits and action logs
- Platform-specific adapters

## Safety Principles

- Treat email addresses and phone numbers as sensitive data.
- Run live simulation only with the platform owner's authorization.
- Do not allow bots to impersonate real people; disclose bot use where end users are affected.
- Validate training material and configuration inputs to reduce prompt-injection and persona-manipulation risks.
- Require client approval for destructive, financial, privacy-sensitive, irreversible, external-publish, or real-user-contact actions.
- Give every integration only the permissions it needs.
- Log agent requests, tool calls, approvals, and outcomes without exposing secrets.
- Provide a clear way to pause, disable, or delete a bot.

## Technical Direction

The backend is built with Go and Postgres.

Current layout:

```text
anarchy/
├── cmd/
│   ├── server/             # Training/export CLI (migrate, import, randomize, export)
│   └── seed/               # Bulk identity JSON generator (local or randomuser.me source)
├── internal/               # Private application packages
│   ├── identity/           # Bot identities, encrypted contact store
│   ├── character/          # Behavior configuration and versioning
│   ├── training/           # Identity x character randomization
│   ├── evaluation/         # Scenarios, scoring, and results
│   ├── integrations/       # Agent-platform export and permissions
│   ├── execution/          # Scheduled tasks and approval gate
│   ├── auth/               # Authentication and authorization
│   └── platform/           # DB connection, AES-256-GCM contact encryption
├── migrations/             # Database migrations (0001 init, 0002 age)
├── functions/              # Identity/character JSON, generated bulk data, exports
├── docs/                   # Architecture and product documentation
├── docker-compose.yml      # Local Postgres
├── go.mod
└── README.md
```

## MVP Scope

1. Create bot identities (bulk JSON seed, Postgres-backed, contacts encrypted).
2. Define and version behavior.
3. Randomize identity x character deployments at volume.
4. Run behavior evaluation scenarios.
5. Review the results and revise the behavior.
6. Export the approved configuration for the owner's agent.
7. Pilot with owner-operated startups before public launch.

## Project Status

Core training pipeline is live: Go service, Postgres with encrypted contacts, bulk JSON seeding, randomized deployments, rule-based evaluation, and versioned agent export. Schedules with jitter, the full task contract, and platform-specific adapters come next. Pilots will run on owner-operated startups first.
