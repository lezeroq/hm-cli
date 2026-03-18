# hm CLI — Design Spec

**Date:** 2026-03-18
**Status:** Approved

---

## Overview

`hm` is a Go CLI tool that takes a natural language description of a shell command and returns the command via Claude. The user can copy the result to clipboard, refine it with a follow-up prompt, or dismiss it. Designed for power terminal users who work across multiple terminal sessions, including remote SSH sessions.

---

## Architecture

Four components with clear, single responsibilities:

### 1. `config`
- Reads and writes `~/.config/hm/config.toml`
- Stores: `clipboard_cmd`, `system_prompt`, `session_id`
- Auto-creates the file with defaults on first run
- `session_id` is managed automatically (written after first Claude call, cleared on `--refresh`)

### 2. `session`
- Manages the dedicated, persistent Claude session
- On first run: invokes `claude` without `--session-id`, captures and persists the returned session ID
- On subsequent runs: passes `--session-id <id>` to `claude`
- `hm --refresh`: clears stored session ID (next run starts a fresh session)
- If a stored session ID is stale/invalid: auto-clears and retries once, saves new ID

### 3. `claude`
- Thin wrapper around `os/exec` invoking the `claude` CLI binary
- Builds full prompt: `<system_prompt>\nUser request: <query>`
- Passes `--session-id` when available
- Captures stdout as the raw command string
- Returns stderr on non-zero exit for error display

### 4. `ui`
- Bubble Tea application
- Renders the command and keybinding hints
- Handles three modes:
  - **Normal**: display command + hints
  - **Refine**: text input for follow-up prompt to Claude
  - **Done**: print command to terminal stdout (for shell history visibility) and exit
- On exit (any path), the command is always printed to terminal so it remains visible in scrollback

---

## Data Flow

```
$ hm "<query>"
      │
      ▼
Load ~/.config/hm/config.toml
      │  session_id present? → use it
      │  missing? → first run, omit --session-id
      ▼
exec: claude -p "<system_prompt>\nUser request: <query>" [--session-id <id>]
      │  capture stdout → raw command string
      │  capture session ID → persist to config
      ▼
Bubble Tea UI:
      ┌──────────────────────────────────────────────────────────┐
      │  kubectl get pods -A --sort-by=.metadata.creationTimestamp  │
      │                                                          │
      │  [enter] copy   [e] refine   [esc/q/n] cancel           │
      └──────────────────────────────────────────────────────────┘
      │
      ├── Enter   → pipe command to clipboard_cmd → print command → exit
      ├── e       → show textinput → user types follow-up
      │             → re-invoke claude with follow-up in same session
      │             → re-render with updated command
      └── Esc/q/n → print command → exit (no clipboard)
```

---

## CLI Interface

```
hm "<query>"              # standard usage
hm --refresh "<query>"   # reset session, then run query
hm --refresh             # reset session only, no query
hm --no-session "<query>"  # stateless one-off call
hm --version             # print version
```

---

## Configuration

File: `~/.config/hm/config.toml`

```toml
clipboard_cmd = "xclip -selection clipboard"
system_prompt = "You are a shell command assistant. Return only the raw shell command, no explanation, no markdown, no code fences."
# session_id managed automatically
session_id = ""
```

The user can extend `system_prompt` with personal context (e.g. default cluster, namespace, preferred tools) to improve command quality over time without repeating it in every query.

---

## Error Handling

| Condition | Behavior |
|-----------|----------|
| `claude` CLI not found | Print: `"hm requires the claude CLI. Install from https://claude.ai/code"` → exit 1 |
| `claude` returns empty output | Show `"No command returned. Try rephrasing."` in UI, remain interactive so user can press `e` |
| `claude` exits non-zero | Surface stderr to user → exit 1 |
| `xclip` (or configured cmd) not found | Print command with note: `"(clipboard unavailable — command printed above)"` |
| Config file missing | Auto-create with defaults silently |
| Stale/invalid session ID | Clear session ID, retry once without it, persist new ID |

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/bubbles/textinput` | Follow-up prompt input field |
| `github.com/BurntSushi/toml` | Config file parsing |
| `golang.org/x/term` | Raw terminal mode (used by bubbletea) |

---

## Out of Scope

- Windows / macOS support (Linux/X11 primary target; macOS clipboard is a config change)
- Web UI or daemon process
- Command history / search across past `hm` queries
- Multiple named sessions
