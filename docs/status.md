# Status - 2026-10-22

Done:
- Full build per docs/spec.md except the warden/tribunal risk design (v1 relays requests to the person; see risk.md).
- Core (~3k lines), Linux namespace sandbox (+WSL, +Seatbelt), Claude Code + fake harness, all default modules, daemon + API, CLI, TUI, first run, autostart, self-update/self-release, installers, CI + release workflows, demo project.
- Tests passing:
  - unit: gate guarantees (pinned merge, check + verdict bound to sha, foreign git metadata refused, only hive/* pushed, protected paths), journal, harness parsing, egress filtering, schedules, TUI fits 50×16 to 120×40;
  - e2e (real daemon, fake harness, local forge): ask→merge with review, projects, failing tests stop with PR open, score drop → revert + lesson quoted in next brief, sweep files an ask, kill/restart at every stage with no duplicates, usage-limit park/resume, crash retry, request → person → agent, module install + update, self-release. Passes both bare and inside the WSL sandbox.
  - WSL sandbox isolation checked by hand: /mnt, /home, WSL interop hidden; /etc, /usr read-only; only checkout writable; public HTTPS via proxy 200; local services 403; no direct network.

Not verified:
- Live agents with real Claude Code (no plan usage spent yet). Sandboxed live runs read a short-lived token from Claude Code's credential file at run time; not exercised.
- The GitHub forge path (gh issue/PR/merge) end to end; only the local forge is exercised.
- Installers and self-update with real release assets (no repo/release published).
- macOS Seatbelt sandbox; native Linux host (same namespace code as WSL).
- Interactive first run and TUI driven by hand (layout test only); autostart registration not executed on this machine.

Next:
1. Owner decisions: publish the repo + first release (makes install lines work); live smoke run with real Claude (plan usage; bare vs sandboxed).
2. Risk module: warden/tribunal per docs/risk.md (owner).
3. Codex adapter.

Resume: read CLAUDE.md, then this file; run `go test -short ./...` and `go test ./internal/e2e/`.
