# Shared prompt replay fixture

`prompt-options-replay.json` preserves the nine-event shape observed in the
2026-09-06 native prompt options incident: an untagged submitted user receipt,
followed by native assistant snapshot/delta/completion and a completed turn.
All user/assistant text, native/session/turn/item identifiers and timestamps
are replaced with fixed fixture values. No workspace, authentication, native
transcript or raw RPC envelope is included.

The Go tests check new typed receipts against the same fixture and reopen an
isolated journal to verify retry identity without starting a native client.
The Web tests exercise normalization, incremental presentation, duplicate
replay and rejection of malformed/unknown metadata. The fixture is an
XGC-owned regression record, not an upstream T3 artifact.
