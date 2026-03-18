# hm CLI — Design Spec

**Date:** 2026-03-18
**Status:** Approved

---

## Overview

`hm` is a Go CLI tool that takes a natural language description of a shell command and returns the command via Claude. The user can copy the result to clipboard, refine it with a follow-up prompt, or dismiss it. Designed for power terminal users who work across multiple terminal sessions, including remote SSH sessions where a local terminal runs `hm` and the user pastes the result into the remote shell.

---

## Architecture

Four components with clear, single responsibilities:

### 1. `config`
- Reads and writes `~/.config/hm/config.toml`
- On first run: creates `~/.config/hm/` directory (and all parents) and the config file with defaults, silently
- Stores: `clipboard_cmd`, `system_prompt`, `session_id`
- `session_id` is managed automatically (written after each successful Claude call, cleared on `--refresh`)
- Users edit `config.toml` directly; no `hm config` subcommand is provided
- If writing `session_id` to config fails (e.g. permission error), print a warning to stderr (`hm: warning: could not save session ID: <err>`) and continue — the tool still works, the session just won't persist

### 2. `session`
- Manages the dedicated, persistent Claude session
- On first run (no `session_id` in config): invokes `claude` without `--session-id`, parses the `session_id` field from JSON output, persists it to config
- On subsequent runs: passes `--session-id <uuid>` to `claude`
- **Stale session handling:** A failed Claude call (non-zero exit OR `is_error: true` in JSON) is treated identically:
  - If a `session_id` was passed: clear the stored `session_id` and retry once with the same arguments but without `--session-id`. If the retry also fails (non-zero exit OR `is_error: true`), surface the error and exit 1.
  - If no `session_id` was passed: surface the error and exit 1 immediately (no retry).
- `hm --refresh`: clears `session_id` in config, prints `"Session cleared."`, exits 0 (if no query given)
- `hm --refresh "<query>"`: clears session ID, then runs query — new session ID is captured and persisted in the same invocation
- `hm --refresh --no-session "<query>"`: `--refresh` still clears the stored `session_id`, then the query runs without a session_id, and the result session_id is discarded (consistent with `--no-session` semantics)

### 3. `claude`
- Thin wrapper around `os/exec`
- Exact invocation (both initial query and follow-up refine calls):
  ```
  claude -p "<query>" \
    --output-format json \
    --system-prompt "<system_prompt from config>" \
    [--session-id <uuid>]        # omitted on first run and --no-session
  ```
  `--system-prompt` is passed on every call. The session retains context between turns, but re-sending the system prompt ensures consistent behavior if Claude's session context is compressed.
- JSON response shape (from `--output-format json`):
  ```json
  {
    "result": "<raw shell command or multi-line string>",
    "session_id": "<uuid>",
    "is_error": false,
    ...
  }
  ```
- `result` may contain newlines (e.g. pipelines, here-docs); rendered as-is in the UI
- Extracts `result` as the command string and `session_id` for persistence
- Error: non-zero exit or `is_error: true` — both treated identically as failure

### 4. `ui`
- Bubble Tea application using `charmbracelet/bubbletea`, `charmbracelet/bubbles/textinput`, `charmbracelet/bubbles/spinner`, and `charmbracelet/lipgloss`
- **Before TUI renders:** a spinner is printed to stderr to indicate the initial Claude call is in progress. This spinner is not a Bubble Tea model — it is a simple character printed to stderr that is overwritten when the TUI starts.
- Three modes:
  - **Normal**: display command + keybinding hints. Long commands soft-wrap within the display box; multi-line results render as-is.
  - **Refine**: `bubbles/textinput` for follow-up prompt, submitted with Enter; shows a `bubbles/spinner` while waiting for Claude; on error, displays the error inline and returns to Normal mode with the previous command
  - **Done**: TUI tears down, then command is printed to stdout (for scrollback visibility), then clipboard is attempted
- Terminal dimensions are obtained via `tea.WindowSizeMsg` from Bubble Tea; no extra terminal library needed

**Exit sequence (Enter pressed):**
1. TUI tears down (restores terminal)
2. Command printed to stdout
3. `clipboard_cmd` invoked via `sh -c "<clipboard_cmd>"` with command piped to stdin
4. If clipboard succeeds: exit 0 silently
5. If clipboard fails: print `"(clipboard unavailable — command printed above)"` to stderr, exit 0

**Exit sequence (Esc/q/n pressed):**
1. TUI tears down
2. Command printed to stdout
3. Exit 0 (no clipboard)

---

## Data Flow

```
$ hm "<query>"
      │
      ▼
Load ~/.config/hm/config.toml (create dir + file with defaults if missing)
      │  session_id present? → use it
      │  missing? → first run, omit --session-id
      ▼
Print spinner to stderr ("Thinking...")
      ▼
exec: claude -p "<query>" --output-format json --system-prompt "<system_prompt>" [--session-id <uuid>]
      │  parse JSON → extract "result" (command), "session_id"
      │  persist session_id to config (warn on failure, continue)
      ▼
Bubble Tea UI starts (clears spinner):
      ┌──────────────────────────────────────────────────────────┐
      │  kubectl get pods -A --sort-by=.metadata.creationTimestamp  │
      │                                                          │
      │  [enter] copy   [e] refine   [esc/q/n] dismiss          │
      └──────────────────────────────────────────────────────────┘
      │
      ├── Enter   → teardown TUI → print command → run clipboard_cmd → exit 0
      ├── e       → show textinput → user types follow-up message
      │             → show spinner → exec claude with follow-up in same session
      │             → on success: re-render Normal mode with updated command
      │             → on error: display error inline, return to Normal mode
      └── Esc/q/n → teardown TUI → print command → exit 0 (no clipboard)
```

**Follow-up prompt:** the follow-up message is sent as a bare `-p` argument with `--session-id`. The session context retains the previous exchange so no re-sending of the original query is needed. `--system-prompt` is included on every call.

**Empty output:** if `result` is empty or whitespace, the UI renders `"No command returned — press [e] to refine or [esc] to dismiss"` in the command area. Enter is disabled in this state.

---

## CLI Interface

```
hm "<query>"                       # standard usage
hm --refresh "<query>"            # reset session, then run query; new session_id is persisted
hm --refresh                      # reset session only; prints "Session cleared." and exits 0
hm --no-session "<query>"         # stateless: omit --session-id, discard returned session_id
hm --refresh --no-session "<query>" # clears session, runs stateless query
hm --version                      # prints "hm v<version>" (version injected at build time via ldflags)
hm                                # no arguments: print usage to stderr and exit 1
```

---

## Error Output Format

- All pre-TUI errors (binary not found, Claude failure, config errors): printed to **stderr** with prefix `hm: <message>`, then exit 1
- Pre-TUI warnings (e.g. failed to save session_id): printed to **stderr** with prefix `hm: warning: <message>`, execution continues
- In-TUI errors (refine call failure): rendered inline in the TUI display area, no exit
- Post-TUI errors (clipboard failure): printed to **stderr** after TUI teardown, exit 0

---

## Configuration

File: `~/.config/hm/config.toml`

```toml
clipboard_cmd = "xclip -selection clipboard"
system_prompt = "You are a shell command assistant. Return only the raw shell command, no explanation, no markdown, no code fences."
# session_id is managed automatically — do not edit manually
session_id = ""
```

**Clipboard command execution:** `clipboard_cmd` is passed to `sh -c "<clipboard_cmd>"` with the command piped to stdin. This allows multi-word arguments and shell features without manual tokenization. The value is user-controlled in their own config file, so shell execution is acceptable.

The user can extend `system_prompt` with personal context (e.g. default cluster, namespace, preferred tools) to improve command quality across all queries without repeating it each time.

---

## Error Handling

| Condition | Behavior |
|-----------|----------|
| `hm` run with no arguments | Print usage to stderr, exit 1 |
| `claude` CLI not found in PATH | `hm: claude CLI not found. Install from https://claude.ai/code` → exit 1 |
| `claude` exits non-zero or `is_error: true`, no session_id passed | Surface error to stderr → exit 1 |
| `claude` exits non-zero or `is_error: true`, session_id was passed | Clear `session_id`, retry without it; if retry also fails, surface error → exit 1 |
| `result` is empty/whitespace | Show `"No command returned — press [e] to refine or [esc] to dismiss"` in UI; Enter disabled |
| Follow-up refine call fails | Display error inline in TUI, return to Normal mode with previous command |
| Config directory missing | Auto-create with `MkdirAll`, silently |
| Config file missing | Auto-create with defaults, silently |
| Writing session_id to config fails | `hm: warning: could not save session ID: <err>` to stderr, continue |
| Clipboard cmd fails at runtime | Print command then `"(clipboard unavailable — command printed above)"` to stderr, exit 0 |
| `--no-session` invocation | Omit `--session-id`; discard any `session_id` returned in response |
| `--refresh` with no query | Clear `session_id` in config, print `"Session cleared."` → exit 0 |
| `--refresh --no-session "<query>"` | Clear `session_id`, run query without session, discard result session_id |
| Concurrent `hm` invocations | Last write wins for `session_id`; no file locking. Accepted behavior — concurrent usage is rare and the worst outcome is a mismatched session ID corrected by stale-session retry |

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework; provides `WindowSizeMsg` for terminal dimensions |
| `github.com/charmbracelet/bubbles/textinput` | Follow-up prompt input field |
| `github.com/charmbracelet/bubbles/spinner` | Spinner while waiting for Claude in Refine mode |
| `github.com/charmbracelet/lipgloss` | Terminal styling for command display box |
| `github.com/BurntSushi/toml` | Config file parsing |

---

## Out of Scope

- Windows / macOS support (Linux/X11 primary target; macOS is a one-line config change: `clipboard_cmd = "pbcopy"`)
- Web UI or daemon process
- Command history / search across past `hm` queries
- Multiple named sessions
- `hm config` subcommand (users edit `config.toml` directly)
- File locking for concurrent config access
