package core

// The hard ceiling. Everything inside it is judgement; these are limits, and
// each is enforced by structure rather than by an agent's good behaviour:
//
//   - Agents hold no credentials. Logins stay with the effects process and the
//     harnesses' own stores; an agent gets only a short-lived token (Tokens).
//   - Agents run sandboxed: they write only their checkout and scratch home and
//     reach the network outbound only (package sandbox).
//   - Main changes only through the gate: a merge pinned to a commit with a
//     host-run passing test and a bound verdict (Effects.Merge). Only hive/*
//     branches are ever pushed.
//   - Goals, the score and the project's settings are the person's to merge
//     (Effects.Classify marks them protected).
//   - The core has no path that runs an agent's request outside the sandbox.
//     A request past an agent's reach becomes a question for the person
//     (Person); the risk module records it and relays the answer.
//   - The records are append-only: the journal, chats, scores and run output
//     live where agents can't write.
//
// AlwaysPerson names what is always the person's, whatever the autonomy
// setting says.
var AlwaysPerson = map[string]string{
	"money":      "spends real money",
	"credential": "mints or rotates a credential",
	"open":       "opens this machine to the network",
	"outward":    "sends something outward in your name",
}
