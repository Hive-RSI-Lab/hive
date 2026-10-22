# The Hive

A small team of AI agents that lives in a folder on your computer and works on your projects while you do other things.

You talk to a lead in a chat. The lead turns what you say into briefs. Workers build each piece in their own sandboxed checkout. Tests run where the agent can't touch them, a reviewer reads the exact commit, and the change merges on GitHub pinned to the commit that was checked. After every merge the Hive measures the project against its score and keeps the change or reverts it.

## Install

macOS / Linux:

    curl -fsSL https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/install.sh | sh

Windows (PowerShell):

    irm https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/install.ps1 | iex

Then run `hive`. It checks for git, Claude Code and the GitHub CLI, offers their own logins, asks where to live (default `~/hive`) and for a GitHub repository (blank makes a small demo project), and opens the control center with the lead's chat.

On Windows, agents are sandboxed inside WSL 2 with Claude Code installed there (`hive doctor` says what's missing).

## Use

    hive                  control center: Chats, Dashboard, Waiting, Modules, Settings
    hive ask "..." --wait ask the lead from a shell
    hive status           what it's doing
    hive improve          run an improvement sweep now
    hive doctor           check the install and sandbox
    hive update           newest release
    hive shutdown         stop it (agents finish their turn; `hive` starts it again)

The Hive keeps running in the background and starts with the machine. Closing the control center never stops work.

## What's safe and what isn't

- Agents hold no logins; they get a short-lived token per turn.
- Agents write only their checkout and a scratch home; network is outbound only, never to this machine or your LAN.
- Main changes only through the gate. Goals, the score and project settings are yours to merge.
- Anything an agent needs beyond its sandbox comes to you in **Waiting**; the Hive doesn't run it for them.

See [docs/architecture.md](docs/architecture.md) and [docs/risk.md](docs/risk.md).

## Uninstall

    curl -fsSL https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/uninstall.sh | sh
    irm https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/uninstall.ps1 | iex

Your folder is left alone.
