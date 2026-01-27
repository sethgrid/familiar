# Prompt Familiar

A terminal-based pet system - a familiar that lives in your terminal that enjoys interaction. Based on a growing catalogue of pets, familiars can be stored per project or directory. One fun use is to have familiars nudge project contributors to critical project updates.


### Example: Project Familiar + Prompt + Messages

Add a tiny familiar to your prompt (after `go install ./cmd/familiar`):

```bash
export PS1='$(familiar health) \w$ '
```

If you're like me and using p10k theme for zsh:

```bash
# ~/.p10k.zsh (or ~/.config/p10k/.p10k.zsh depending on your setup)

function prompt_familiar_health() {
  local val
  val=$(familiar health 2>/dev/null)
  [[ -n $val ]] && p10k segment -f yellow -t "$val"
}

# elsewhere in the file:
typeset -g POWERLEVEL9K_LEFT_PROMPT_ELEMENTS=(
  #...
  familiar_health
  #...
)
```

You're working in a shared repo:

```
~/code/sharedproject 🐾
git pull origin main
# ...
# updating configs...

~/code/sharedproject 💬
```

Check what the familiar is trying to say:

```
~/code/sharedproject 💬
familiar status

Pip
has-message

 /\_/\ 
( o.o )
 > ^ <*

Message: Attn Devs — new local config defaults available.
~/code/sharedproject 💬
familiar acknowledge

```
Pip
happy

 /\_/\ 
( o.o )
 > ^ <*
Pip feels acknowledged

~/code/sharedproject 🐾
```

## Features

- **Hierarchical pet discovery**: Automatically finds familiars in your project directory or uses a global familiar
- **Two-file TOML structure**: Static config (`pet.toml`) and dynamic state (`pet.state.toml`)
- **Derived state system**: Health and conditions computed at runtime
- **ASCII art rendering**: Beautiful terminal art for your familiar
- **CLI-driven decay**: No background daemon - decay only happens when you interact
- **Multiple states**: Happy, hungry, tired, sad, lonely, infirm, stone, and has-message

## Installation

```bash
go build ./cmd/familiar
sudo mv familiar /usr/local/bin/
```

## Quick Start

### Initialize a Familiar

```bash
# Create a project-local familiar
familiar init MyCat

# Create a global familiar
familiar init --global MyCat
```

### Check Status

**Default (concise):**
```bash
familiar status
```

Shows:
- One short state summary line
- ASCII art / animation
- Optional message if present

**Verbose (stats card):**
```bash
familiar status -v
# or
familiar status --verbose
```

Shows:
- Same art/animation as default
- Stats card above the art:
  - Health (derived)
  - Hunger / Happiness / Energy
  - Evolution
  - Flags like stone / infirm / lonely / hungry

### Interact with Your Familiar

```bash
familiar feed      # Feed your familiar
familiar play      # Play with your familiar
familiar rest      # Let your familiar rest
familiar message "ship is red"  # Set a message
familiar acknowledge  # Acknowledge your familiar
```

**Acknowledge Behavior:**

Normal mode:
```bash
familiar acknowledge
```
- Clears the message
- Updates mood/energy a bit
- Shows happy art (regardless of previous state)
- Displays confirmation message

Silent mode (for scripts/hooks):
```bash
familiar acknowledge -s
# or
familiar acknowledge --silent
```
- Same state changes (clears message, bumps happiness)
- No output (just exit 0)
- Useful for scripts / hooks that don't want to spam stdout

### Prompt Integration

Add to your shell prompt (e.g., in `~/.bashrc` or `~/.zshrc`):

```bash
export PS1='$(familiar health) \w$ '
```

This will show:
- 🐾 when your familiar is healthy and happy
- 💬 when your familiar has a message waiting
- Other indicators based on your familiar's state

## ASCII Cat Familiar

The default familiar is an ASCII cat with different states:

- **Default**: `/\_/\` `( o.o )` `> ^ <`
- **Infirm**: `/\_/\` `( x.x )` `> ^ <`
- **Stone**: `/\_/\` `( +.+ )` `> ^ <`
- **Egg**: `___` `/  . . \` `\___/`

## Project Structure

```
familiar/
├── cmd/familiar/          # CLI entry point
├── internal/
│   ├── pet/              # Pet models (config, state, decay)
│   ├── conditions/       # Derived conditions system
│   ├── health/           # Health computation
│   ├── discovery/        # Pet discovery logic
│   ├── art/              # ASCII art rendering
│   └── storage/          # TOML storage layer
└── integration_test.go   # Integration tests
```

## Development

Run tests:

```bash
go test -v ./...
```

Build:

```bash
go build ./cmd/familiar
```

## License

MIT
