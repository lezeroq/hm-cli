# hm

A small Linux terminal helper that turns natural language into shell commands.
Type what you want, get a command, copy it to clipboard.

Built for personal convenience — I use i3/sway and spend most of my time in
terminals. I didn't want to switch context or open a browser just to look up a
command. Inspired by [tldr](https://tldr.sh): quick, to the point, stays in the
terminal.

Built with the help of Claude. Nothing unique here — it just shells out to the
[Claude Code CLI](https://claude.ai/code) and reuses your existing subscription.
Each query is a message in a persistent Claude session, so follow-up questions
have context from previous ones.

## Install

```bash
git clone <repo>
cd hm
make install   # copies hm and hmq to ~/.local/bin
```

Requires the `claude` CLI in your PATH and `xclip` for clipboard (configurable).

## Usage

```
hm "get pods from all namespaces sorted by age"
```

A small TUI appears with the generated command. Press `Enter` to copy to
clipboard, `e` to refine with a follow-up, `Esc`/`q` to dismiss.

```
hm --examples   # short cheatsheet
hm --help       # all options
```

## Config

`~/.config/hm/config.toml` is created on first run. You can change the
clipboard command (e.g. `pbcopy` on macOS) and the system prompt there.
