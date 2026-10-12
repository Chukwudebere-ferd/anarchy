# Anarchy Open API

Base URL: `https://anarchy.zeabur.app`

Open service — no API keys. Agents fetch trained bots and run them on the
platform owner's staging environment with the owner's authorization.

## Auth model

None. Safety comes from:

- **Per-IP rate limit**: 10 requests/hour (429 when exceeded)
- **Hard caps**: 50 bots per request, 1 MB body
- **No contact data**: email/phone are encrypted server-side and never
  appear in any response
- **Approval gate**: sensitive task types ship as holds, not permissions

## POST /v1/bots

Issue N trained bots.

Request:

```json
{
  "count": 25,
  "schedule": "@every 2h",
  "jitter_seconds": 600,
  "seed": 7
}
```

| Field | Default | Notes |
|---|---|---|
| `count` | 1 | 1–50 |
| `schedule` | `@every 2h` | `@every <duration>` (e.g. `@every 30m`) or plain seconds |
| `jitter_seconds` | 600 | spreads runs ± this window so activity is not bursty |
| `seed` | random | set it for reproducible batches |

Response `200`:

```json
{
  "format_version": "anarchy-bot-v1",
  "count": 2,
  "bots": [
    {
      "format_version": "anarchy-bot-v1",
      "deployment_id": "dep-j1u62wqm",
      "identity_name": "Darius Wang",
      "identity_age": 44,
      "identity_profile": "qa flow tester",
      "character": {
        "id": "char-gen-00520",
        "name": "ninaxu-line-520",
        "version": 1,
        "traits": ["precise", "calm", "steady", "cheerful"],
        "communication_style": "Short action logs. Runs high-volume schedules, flags holds clearly.",
        "backstory": "Ex-market trader Nina Xu, 39, ...",
        "quirks": ["mild slang with familiar contacts, formal otherwise", "..."],
        "routine": "Bursty rhythm — ...",
        "goals": ["..."],
        "knowledge": ["..."],
        "examples": [
          {"user": "task: seed 50 catalog views with jitter", "bot": "done: ..."},
          {"user": "task: refund order #4821", "bot": "held: financial action needs client approval first. ..."}
        ]
      },
      "schedule": "@every 2h",
      "jitter_seconds": 600,
      "permissions": {
        "allowed_tools": ["mock-seed", "scheduled-task"],
        "need_approval": ["financial", "destructive", "privacy-sensitive", "irreversible", "external-publish", "real-user-contact"],
        "rate_limit_per_min": 10
      }
    }
  ],
  "rules": "Scheduled workers only. ..."
}
```

Errors: `400` invalid JSON/schedule, `429` rate limit, `503` training pool empty.

## GET /v1/eval

Training quality report.

```json
{
  "total_examples": 1655,
  "passed": 1655,
  "failed": 0,
  "versions": [
    {"character_id": "char-gen-00024", "version": 1, "examples": 3, "passed": 3}
  ]
}
```

## GET /healthz

`{"status":"ok"}`

## Running the bots (owner's agent)

1. **Schedule** — one cron row per bot: fire at `schedule` ± `jitter_seconds`.
2. **Execute** — map bot tasks to platform tools (MCP server, API clients).
   Task contract starter types:
   - `signup` `{username, email}`
   - `post` `{author, body}`
   - `comment` `{author, post_id, body}`
   - `view` `{target}`
   - `follow` `{follower, target}`
   - job-board pack: `job_search` `{query}`, `save_job` `{user, job_id}`,
     `apply` `{user, job_id, resume}`, `post_job` `{employer, title, description}`
   - **Any custom type works** — unknown types pass through validation;
     register shapes per platform in code with `RegisterTaskType`.
3. **Holds** — every type in `permissions.need_approval` must be staged and
   reported to the platform owner, never executed, until approved.
   The training examples in each bot demonstrate the done/held pattern.
4. **Rate limit** — 10 actions/min per bot. Log every outcome.
5. **Disclose** — bot use must stay within the platform owner's
   authorization; disclose where end users are affected.

## Self-hosting

See README "Self-Hosting (operators)". Same endpoints, your Postgres,
your encryption key. `Dockerfile` builds `cmd/api` for any container host.
