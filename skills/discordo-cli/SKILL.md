---
name: discordo-cli
description: Read Discord guilds, channels, message history, threads, members, pins and DMs from the terminal with the read-only `discordo` CLI. Use when an agent needs Discord context without a bot token, when reading a specific channel or message, when searching a guild's messages, or when the TUI is already logged in and only JSON output is needed.
---

# discordo CLI

## Overview

`discordo` is a Discord terminal client. Run with no arguments it starts the TUI.
With a subcommand it behaves as a **read-only CLI** that prints JSON, reuses the
token the TUI stored in the OS keyring, and never opens a gateway connection.

The CLI uses the same browser headers and TLS fingerprint as the TUI, so it
speaks to Discord as the logged-in user account.

## Preconditions

1. Log in once through the TUI.
- Run `discordo` and complete the login (QR login is the most reliable path,
  since the password path does not support MFA).
- The token is written to the OS keyring, not to `config.toml`.

2. Verify the session.
- `discordo auth status`
- If it returns `"reason": "no_token"`, the TUI login has not happened on this
  machine, or `DISCORDO_TOKEN` is stale.

3. Override the token only when needed.
- `DISCORDO_TOKEN=<token> discordo guild list` uses the environment token and
  never persists it.
- Prefer the keyring path; never write tokens into repositories or logs.

## Workflow

1. Resolve names to IDs once, then reuse the IDs.
- `discordo guild list` → guild IDs.
- `discordo channel list <guild-id>` → channel IDs.
- Cache them in the agent's own context; the CLI is stateless.

2. Read the narrowest thing that answers the question.
- Recent channel context: `discordo msg list <channel-id> -limit 50`
- One known message: `discordo msg get <channel-id> <message-id>`
- Keyword lookup across a guild: `discordo search <guild-id> -query "..."`
- Follow-ups: `discordo thread list <channel-id>`

3. Page backwards with the returned cursor.
- `msg list` returns `next_before` when the result hit `-limit`.
- Pass it back: `discordo msg list <channel-id> -before <next_before>`

4. Verify the result shape.
- Success: `{"status": "ok", "<data-key>": ..., "count": N}`
- Failure: JSON on stderr with `status`, `reason`, `error`, optional `code`.

## Output contract

- JSON on stdout, two-space indented, always (no table mode).
- Messages are ordered **oldest to newest**. The REST API returns them newest
  first; the CLI reverses them.
- Snowflake IDs are strings. Timestamps are RFC 3339.
- Message objects are normalized, not raw API payloads: `id`, `channel_id`,
  `author`, `timestamp`, `content`, `attachments`, `reactions`, `reference`,
  `embed_count`, `url`.
- Attachments are metadata only (`filename`, `size`, `content_type`, `url`).
  The CLI never downloads files.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | failure |
| 2 | usage error |
| 3 | authentication (no token or 401) |
| 4 | Discord API error (403/404/429/other) |

## Flags

- Flags may appear before or after positional arguments.
- `-limit` is capped per command; passing an out-of-range value is a usage error.
- `-before` and `-after` are mutually exclusive.

## Command Reference

Load `references/commands.md` for copy-paste recipes: guild/channel listing,
message history, message lookup, search filters, threads, members, pins, DMs,
and account status.

## Constraints

- Read-only by design. The CLI has no send, edit, delete, or reaction commands.
- Some endpoints are bot-only and are deliberately absent: listing all guild
  members, searching members, and fetching the active threads of a whole guild.
  `member get` works for one known user; threads are listed per channel with
  `thread list`.
- Search requires a guild ID: `discordo search <guild-id> -query ...`. There is
  no cross-guild search, because the user search endpoint is guild-scoped.
- No gateway connection, so no realtime events and no presence data.
- Automation of a user account is self-bot behaviour under Discord's terms.
  Keep request volume low and avoid bulk loops.
- Respect 429 responses: the CLI reports `"reason": "rate_limited"`; wait before
  retrying instead of looping.
