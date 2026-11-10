# discordo CLI command reference

All commands print JSON on stdout. Errors are JSON on stderr with a non-zero
exit code. Replace `<...>` placeholders with snowflake IDs from a previous call.

## Account

```sh
discordo auth status
printf '%s' "$TOKEN" | discordo auth login   # validate and store a token
```

```json
{
  "status": "ok",
  "account": {"id": "...", "username": "...", "global_name": "..."},
  "token_source": "keyring"
}
```

## Guilds

```sh
discordo guild list
discordo guild list -limit 20
discordo guild get <guild-id>
```

`guild get` includes `member_count` and `online_count`.

## Channels

```sh
discordo channel list <guild-id>
discordo channel get <channel-id>
```

`type_name` is one of `guild_text`, `guild_voice`, `guild_category`,
`guild_announcement`, `public_thread`, `private_thread`, `guild_forum`,
`guild_stage_voice`, `dm`, `group_dm`.

## Message history

```sh
# most recent 50 messages, oldest to newest
discordo msg list <channel-id>

# explicit window
discordo msg list <channel-id> -limit 100
discordo msg list <channel-id> -before <message-id>
discordo msg list <channel-id> -after <message-id>

# one message (fetched through the list endpoint around the id)
discordo msg get <channel-id> <message-id>
```

When the result hits `-limit`, the response carries `next_before`:

```json
{"status": "ok", "messages": [...], "count": 100, "next_before": "1234..."}
```

Page backwards with `discordo msg list <channel-id> -before <next_before>`.

## Search

Search is guild-scoped and needs at least one filter.

```sh
discordo search <guild-id> -query "deploy failed"
discordo search <guild-id> -query "deploy" -channel <channel-id>
discordo search <guild-id> -author <user-id> -sort desc
discordo search <guild-id> -query "incident" -sort asc -offset 25
```

Options:

| Flag | Meaning |
| --- | --- |
| `-query` | text to match |
| `-channel` | restrict to one channel |
| `-author` | restrict to one author |
| `-sort` | `asc` or `desc` (default `desc`) |
| `-offset` | result offset for paging |

The response includes `total_results` and `offset`. Discord returns results in
pages of 25; advance with `-offset`.

## Threads

Threads are listed per channel. Listing the active threads of a whole guild is a
bot-only endpoint, so it is not available.

```sh
discordo thread list <channel-id>
discordo thread list <channel-id> -query "deploy" -limit 50
discordo thread list <channel-id> -archived
discordo thread archived <channel-id> -limit 50
```

`thread list` returns `has_more` and `total_results`; `thread archived` returns
`has_more`.

## Members

```sh
discordo member get <guild-id> <user-id>
```

Listing or searching guild members requires a bot token, so only single-member
lookups are supported.

## Pins

```sh
discordo pin list <channel-id>
```

## Direct messages

```sh
discordo dm list
```

Returns DM and group DM channels with `recipients`. Read a conversation with the
normal message commands: `discordo msg list <dm-channel-id>`.

## Users

```sh
discordo user me
discordo user get <user-id>
```

## Troubleshooting

1. `"reason": "no_token"` — log in through the TUI first, or set
   `DISCORDO_TOKEN`. The keyring entry is service `discordo`, account `token`.
2. `"reason": "unauthorized"` (HTTP 401) — the stored token was revoked or
   rotated. Log in again in the TUI; the CLI picks the new token up on the next
   run.
3. `"reason": "forbidden"` (HTTP 403) — the account cannot view that channel.
4. `"reason": "not_found"` (HTTP 404) — wrong ID, or the message/channel is not
   visible to this account.
5. `"reason": "rate_limited"` (HTTP 429) — stop and retry later. Do not loop.
6. `"reason": "usage"` — argument problem. Run `discordo help`.
