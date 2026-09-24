# Anarchy

Anarchy is a bot training and deployment platform. Each bot has a defined identity, a configurable character, and measurable behavior. Bots can later be connected to agent platforms such as OpenCode so they can perform approved tasks on external websites.

## Goals

- Create and manage multiple bots.
- Give each bot a structured identity and communication style.
- Train characters using explicit traits, goals, knowledge, boundaries, and examples.
- Evaluate whether a trained bot behaves consistently.
- Export or serve bot configurations for compatible agent platforms.
- Run approved website actions with least-privilege access and an audit trail.

## Core Concepts

### 1. Bot Identity

A bot identity contains its user-facing and operational data, including:

- Name
- Email
- Phone number
- Profile
- Description

Contact details are sensitive data. The platform must encrypt them, restrict access, support deletion, and require a clear basis for their use.

### 2. Character Training

Character training will use structured data instead of a single unstructured prompt. A bot's character can define:

- Personality traits
- Communication style
- Goals and priorities
- Relevant knowledge
- Behavioral boundaries
- Example responses

The objective is consistent, natural communication. The platform should not be used for deceptive impersonation or to bypass another platform's identity and access controls.

### 3. Behavior Evaluation

Training is not complete until the bot can be evaluated. Evaluation scenarios can test:

- Character consistency
- Tone and communication style
- Accuracy and use of available knowledge
- Boundary enforcement
- Task completion
- Appropriate requests for human approval

Each bot should have a versioned training configuration and a history of evaluation results so regressions can be identified.

### 4. Agent Integration

Bot configurations will be adapted for compatible agent platforms such as OpenCode. Personality instructions and tool permissions will remain separate so connecting a bot does not automatically grant unrestricted access.

Agent integrations will support:

- Explicit bot configuration export
- Versioned prompts and character data
- Least-privilege tool permissions
- Human approval for sensitive actions
- Rate limits and action logs
- Platform-specific adapters

The exact OpenCode integration format has not yet been selected or verified.

## Safety Principles

- Treat email addresses and phone numbers as sensitive data.
- Do not allow bots to impersonate real people without explicit authorization and clear disclosure.
- Validate training material and configuration inputs to reduce prompt-injection and persona-manipulation risks.
- Require human approval for destructive, financial, privacy-sensitive, or irreversible actions.
- Give every integration only the permissions it needs.
- Log agent requests, tool calls, approvals, and outcomes without exposing secrets.
- Provide a clear way to pause, disable, or delete a bot.

## Technical Direction

The backend will be built with Go.

A proposed service structure is:

```text
anarchy/
├── cmd/                  # Application entry points
├── internal/             # Private application packages
│   ├── identity/         # Bot identities and contact data
│   ├── character/        # Character configuration and versioning
│   ├── training/         # Training workflows
│   ├── evaluation/       # Scenarios, scoring, and results
│   ├── integrations/     # Agent-platform adapters
│   ├── execution/        # Approved task execution and audit logs
│   ├── auth/             # Authentication and authorization
│   └── platform/         # Configuration and application wiring
├── migrations/           # Database migrations
├── web/                  # Dashboard assets, if used
├── docs/                 # Architecture and product documentation
├── go.mod
└── README.md
```

This structure is a recommendation, not an implemented layout. The final folder structure should be confirmed when the first Go service is scaffolded.

## MVP Scope

The first release should focus on the bot lifecycle before adding external website automation:

1. Create a bot identity.
2. Define and version its character.
3. Run behavior evaluation scenarios.
4. Review the results and revise the character.
5. Export the approved configuration for an agent integration.

Website automation should be added after the configuration format, permissions, approvals, and audit logging are defined.

## Project Status

The project is currently in the planning and requirements stage. No Go services, database schema, agent adapters, or website automation have been implemented yet.
