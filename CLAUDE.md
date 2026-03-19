# CLAUDE.md — contributor guide for AI assistants

## What this project is

`hm` is a Go CLI that turns natural-language queries into shell commands using
the Claude Code CLI (`claude`) as a subprocess. It keeps a persistent Claude
session across invocations so follow-up queries share context.

## Repository layout

```
main.go                      — entry point, flag parsing, orchestration
internal/
  config/   config.go        — TOML config (~/.config/hm/config.toml): clipboard cmd,
            config_test.go     system prompt, session ID, last command
  claude/   client.go        — wraps `claude -p … --output-format json`; ExecFunc is
            client_test.go     injectable so tests never spawn a real subprocess
  clipboard/clipboard.go     — fire-and-forget clipboard via sh -c <cmd>; uses StdinPipe
              clipboard_test.go  so xclip (which stays alive as a server) doesn't block
  ui/       model.go         — Bubble Tea inline TUI (no alt-screen); three modes:
            model_test.go      normal → refine → waiting
Makefile                     — build / vet / test / install targets
```

## Build & test

```bash
make build    # produces ./hm binary
make test     # go vet + go test ./...
make install  # copies binary to ~/.local/bin/hm and creates hmq symlink
```

## Key design decisions

**No API key required** — `hm` shells out to the `claude` CLI (Claude Code) so it
reuses the user's existing subscription. The claude binary must be in PATH.

**Inline TUI, not alt-screen** — Bubble Tea is intentionally run without
`WithAltScreen()`. All output stays in terminal scrollback after the program exits.
Resize handling: widening the terminal re-renders cleanly; narrowing triggers
`tea.ClearScreen` to avoid overlapping-box artifacts (Bubble Tea inline mode tracks
logical newlines, not visual rows).

**Atomic config writes** — `config.write()` writes to a temp file then `os.Rename`s
it into place to avoid corruption on interrupted writes.

**ExecFunc injection** — `claude.Client` accepts an `ExecFunc` so unit tests can
return canned responses without executing the real CLI.

**`hmq` symlink** — `main()` checks `filepath.Base(os.Args[0])`; if it equals `hmq`
it prepends `-q` to the argument list. No second binary needed.

## Adding a new flag

1. Declare a variable in `run()` near the other flag vars.
2. Add a `case` in the `for i := 0; i < len(args)` loop (use index-based loop if
   the flag takes a value, like `--continue`).
3. Update the `usage` const and the `--examples` output in `main.go`.
4. Add a test in `main_test.go` if one exists, or cover the path in integration.

## Config file

`~/.config/hm/config.toml` — created automatically on first run with defaults:

```toml
clipboard_cmd = "xclip -selection clipboard"  # or "pbcopy" on macOS
system_prompt = "You are a shell command assistant. Return only the raw shell command…"
session_id    = ""      # managed automatically; clear with hm --refresh
last_command  = ""      # set after every successful query; used by hm --continue
```
