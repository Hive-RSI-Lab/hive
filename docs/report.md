# Sharing failures with the developer

An install can send its failures and usage to the Hive's developer so the improve loop learns from real installs. It is off unless the person says yes. The installer asks on the terminal (`/dev/tty` on Linux and macOS, so `curl | sh` works; the console on Windows); with no terminal, or on Enter, the answer is no, and `HIVE_SHARE=yes` or `HIVE_SHARE=no` answers ahead of time. The answer goes to `share` in `.hive/config.json` (a time when yes, `off` when no); before first run has made the folder, it waits in `share` next to the workspace pointer file (`~/.config/hive/` on Linux) and first run moves it in. First run asks only if nothing has answered, a reinstall doesn't ask again, and `hive report on|off` or Settings → "share failures" changes it. Nothing is sent from before sharing was turned on.

## What the Hive does

- The `report.send` job (default `every 6h`, a schedule like any other) reads the journal since the last batch it sent, builds one payload, scrubs it, and POSTs it to `share_url` in `.hive/config.json` (default `https://hive.ihsan.cc/api/report`). `hive report send` runs it now.
- A batch with no usage and no failures isn't sent. An unreachable address, a timeout (30 s) or a non-2xx answer leaves the batch for the next run; the Hive carries on either way.
- Every batch sent is kept as `.hive/reports/sent/<id>.json`, byte for byte what went out. The copy is written before the batch leaves and removed if it doesn't arrive; if it can't be written, nothing is sent. `hive report` shows the state; `hive report show` prints the last batch.
- `hive doctor` failures are kept in `.hive/reports/outbox.jsonl` (only while sharing is on) and go with the next batch; one noted while a batch is on its way waits for the one after.
- The install id is a random UUID made when sharing is first turned on, kept in `install_id` in the config. There is no account.

Scrubbed on the machine before anything leaves (`internal/mod/report/scrub.go`): private key blocks, credentials in URLs, known token shapes (GitHub, Anthropic/OpenAI `sk-`, Stripe, Slack, AWS, Google, GitLab, npm, Hugging Face, JWTs), `Bearer`/`Basic` values, whole `Cookie` and `Set-Cookie` header values, anything after a name like `token`, `secret`, `password`, `api_key`, `auth` or `authorization` with `=` or `:`, emails, home folders (`/home/x`, `/Users/x`, `C:\Users\x`, `/root`, the actual home dir → `~`, the Hive folder → `<hive>`), and any run of 32+ letters and digits. No repository contents are sent beyond the failure's own reason and the last 8 lines of the task's log.

## Payload, schema 1

`POST <share_url>`, `Content-Type: application/json`, `User-Agent: hive/<version>`. One JSON object:

| Field | Type | Meaning |
|---|---|---|
| `schema` | int | `1`. A change that breaks a reader bumps it. |
| `install_id` | string | Random UUID v4, stable per install. |
| `sent_at` | RFC 3339 time | When the batch was built. |
| `from`, `to` | RFC 3339 time | First and last journal event from the period, whether or not it was a failure (zero time when the journal had nothing new). |
| `version` | string | Hive version, e.g. `0.1.0` or `0.0.0-dev`. |
| `os`, `arch` | string | Go's `GOOS`, `GOARCH` (`linux`, `darwin`, `windows`; `amd64`, `arm64`). |
| `harness` | string | `claude`, `codex` or `fake`. |
| `usage` | object | Counts in the period, below. |
| `failures` | array | Up to 50, oldest first, below. Never null. |

`usage`: `runs` (agent runs created), `turns` (agent turns ended), `merges` (tasks merged, reverts included), `reverts` (revert tasks merged), `cost_usd` (number, the harness's reported cost; 0 when it reports none). Token counts aren't sent: the harness doesn't report them.

Each failure:

| Field | Type | Meaning |
|---|---|---|
| `kind` | string | `task-blocked` (the worker stopped and said why), `task-stopped` (the Hive gave up: red tests after the rounds, no verdict, conflicts), `run-error` (a turn ended in an error and was parked to retry), `alert` (an alert the Hive raised), `crash` (the daemon started again without having shut down), `doctor` (a `hive doctor` check failed). Readers should keep unknown kinds. |
| `at` | RFC 3339 time | When it happened. |
| `role` | string, optional | The agent's role for `run-error` (`lead`, `worker`, `reviewer`, ...). |
| `text` | string | The reason, scrubbed, at most 2000 characters. |
| `excerpt` | string, optional | Log lines around it, scrubbed, at most 4000 characters. |

Caps: the body is at most 256 KiB; the Hive drops the newest failures until it fits. A receiver may refuse anything larger with 413, and should answer 2xx once it has stored a batch. The Hive doesn't retry a batch it got 2xx for, and resends a batch it didn't, so a receiver may see the same events twice after a lost answer; `install_id` with `from`/`to` identifies a repeat.

Example:

```json
{
  "schema": 1,
  "install_id": "3f0c8a5e-2b7d-4c1e-9a6f-0d4b8e2c7a91",
  "sent_at": "2026-10-22T21:00:00Z",
  "from": "2026-10-22T15:02:11Z",
  "to": "2026-10-22T20:58:40Z",
  "version": "0.1.0",
  "os": "linux",
  "arch": "amd64",
  "harness": "claude",
  "usage": {"runs": 7, "turns": 23, "merges": 2, "reverts": 0, "cost_usd": 1.84},
  "failures": [
    {"kind": "task-stopped", "at": "2026-10-22T18:30:02Z", "text": "stopped after 3 rounds of fixes; what remains: ...", "excerpt": "tests failed in TestParseHeader\n..."}
  ]
}
```
