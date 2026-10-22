# Risk: what v1 does, and the intended design

## v1 behaviour

- Risk scoped to the Hive is the Hive's own business: workers act freely inside their checkout, modules, prompts and the knowledge base. Version control, the append-only journal, and records agents can't write protect it.
- Anything past an agent's sandbox (`hive agent warden --want --why --steps`) becomes an item in the person's **Waiting** queue. The Hive runs nothing on the agent's behalf; the person acts (or declines) and the answer is sent back to the agent as a message.
- Installing a module is the one Hive-performed action behind approval: on "install", the Hive clones the given git URL into `.modules/` (a fixed `git clone`, nothing agent-written).
- Always the person's: money, new credentials, opening the machine to the network, anything outward in their name.

## Intended design (owner's words, not built in v1)

To be implemented by the owner; recorded here so the module can grow into it.

- Risks are weighed by **scope**. Something can look risky but be fine if its effect is confined to the Hive, which version control and protected logs make recoverable.
- Two kinds of risk: risk to the person, including acting against what they wanted (weigh against high-level goals and what's needed to reach them); and risk of interfering with external things.
- **Warden**: privileged over a worker but still bounded; can't do anything fully dangerous. Handles the in-between: actions that reach outside the Hive but are needed and can be scoped (e.g. opening routes into the Hive). Must weigh safety thoroughly before acting.
- **Tribunal**: higher-stakes questions. Researches and votes; does nothing itself. Gets a script to approve, sent by a worker or a warden.
- **The person**: anything the tribunal can't decide. The person can also approve the script sent to the tribunal directly.
- Autonomy setting shifts the lines; never removes the person from what is always theirs.

## Where it plugs in

- `internal/mod/risk`: the `warden` op and the person handlers (`request`, `module`).
- `core.Person`: the queue and answer routing.
- `core.Runs.Wait`: a requesting agent idles (not nudged) until the answer arrives.
- The run sandbox (`internal/sandbox`) is where any wider, scoped grant would live.
