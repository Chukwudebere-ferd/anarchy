# Onboarding a company

How a new startup goes from zero to running bots — three paths, pick by need:

- **API path (fastest)**: their agent calls `https://anarchy.zeabur.app/v1/bots`
  directly. See [api.md](api.md). Good for testing and small batches.
- **Bundle path**: we generate a per-company bundle from the training pool.
  Good for curated pilots and larger rosters.
- **Self-host path**: they run the whole stack. See README "Self-Hosting".

## 0. Worked example (completed pilot)

First pilot ran end-to-end on a live Next.js/Drizzle blog (GracePress-style)
with 30 bots. Steps, reproducible for any company:

1. **Isolation guard** — verify the test DB differs from prod (different
   host) before anything runs. Prod DB is never read by bot code paths.
2. **Migrate test DB only** — run migrations with `DATABASE_URL` forced to
   the test DB; prod env never loaded.
3. **Seed bots** — engine inserts bot users directly
   (`emailVerified: true, isBot: true`): no auth walls, no OTP, no inboxes
   needed. Bots write posts/likes/comments straight to tables.
4. **Run a cycle** — at most 1 post + a few likes + sometimes 1 comment per
   run, per-bot cooldowns, posting-hours window, content screening.
5. **Verify** — count rows by table; confirm prod untouched (read-only count).
6. **Kill switch** — disable → fire the run endpoint → confirm zero new
   activity → re-enable. Missing `CRON_SECRET` was caught and fixed here.

Result: 23 bot posts, 112 likes, 9 comments, 165 activity rows in the test
DB; prod DB untouched; kill switch proven.

## 1. Intake

Collect from the company:

- Platform type and staging URL (never production first)
- 2–3 task types (starter contract or custom — see [api.md](api.md))
- 5–10 sample payloads per task type
- Who approves sensitive holds (name/handle for the `Approve(by)` field)
- Posting rules: rate limits, quiet hours, content boundaries
- Access model: direct DB writes in an isolated test DB (fastest),
  test accounts with whitelisting, or sandbox API keys

## 2. Generate

```sh
anarchy-server.exe --eval
anarchy-server.exe --use-db -n <COUNT> -seed <N> --schedule "@every 2h" --jitter 600 \
  --bundle bundle/<company> --company <company> --format <opencode|generic>
```

- `--eval` must pass before anything ships.
- `--format opencode` (default): `opencode.json`, `anarchy-rules.md`,
  `.opencode/agents/*.md`, `anarchy-bots.json`, wire-up README.
- `--format generic`: framework-neutral — `agents/*.md` system prompts,
  `roster.json`, `rules.md`, README. Works with CrewAI, LangChain,
  AutoGen, or any custom runner.
- One worker file per character version in both formats; the roster maps
  every bot to its worker, identity, schedule and jitter.

## 3. Wire-up (company side, any agent)

OpenCode path:

1. Copy the bundle into their OpenCode project root.
2. Merge `opencode.json` with existing project config.
3. Map their platform tools (MCP server): allow scheduled mock-data tools,
   keep `edit: deny`, sensitive types on `ask`.
4. One cron row per roster entry (`schedule` ± `jitter_seconds`),
   invoking that row's `agent` with the task payload.

Any other agent (generic bundle):

1. Use `agents/<name>.md` as the worker system prompt.
2. Drive scheduling from `roster.json`; enforce `rules.md` holds.
3. Map platform tools the same way: allow mock-data actions,
   deny sensitive by default, human approves holds.

## 4. Pilot review

- Watch the first 50 runs: completion rate, holds raised, realism.
- Holds (`financial`, `external-publish`, `real-user-contact`, …) go to
  the named approver; measure approval turnaround.
- Revise behavior → new character version → re-eval → re-export.
- Promote to wider schedules only after the pilot reads real enough.

## 5. Rules that never bend

- Staging before production, always.
- Platform owner's authorization, in writing.
- No real-person impersonation; disclose bot use where end users are affected.
- No verification-code workarounds on inboxes nobody owns.
- Pause/disable/delete path tested before scale-up.
