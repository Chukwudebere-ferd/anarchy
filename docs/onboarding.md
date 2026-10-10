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
  --bundle bundle/<company> --company <company>
```

- `--eval` must pass before anything ships.
- The bundle contains `opencode.json`, `anarchy-rules.md`,
  `.opencode/agents/*.md` (one worker per character version),
  `anarchy-bots.json` (full roster with schedules), and a wire-up README.

## 3. Wire-up (company side)

1. Copy the bundle into their OpenCode project root.
2. Merge `opencode.json` with existing project config.
3. Map their platform tools (MCP server): allow scheduled mock-data tools,
   keep `edit: deny`, sensitive types on `ask`.
4. One cron row per roster entry (`schedule` ± `jitter_seconds`),
   invoking that row's `agent` with the task payload.

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
