# Onboarding a company

How a new startup goes from zero to running bots, without us in the loop.

## 1. Intake

Collect from the company:

- Platform type and staging URL (never production first)
- 2–3 task types from the contract: `signup`, `post`, `comment`, `view`, `follow`
- 5–10 sample payloads per task type
- Who approves sensitive holds (name/handle for the `Approve(by)` field)
- Posting rules: rate limits, quiet hours, content boundaries
- Email domain decision: their catch-all subdomain, our test domain, or staging inbox service

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
