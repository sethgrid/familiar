# Prompt Familiar

A terminal-based pet system - a familiar that lives in your terminal that enjoys interaction. Based on a growing catalogue of pets, familiars can be stored per project or directory. One fun use is to have familiars nudge project contributors to critical project updates.


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

Pip (evolution 1)
has-message

 /\_/\ 
( o.o )
 > ^ <*

Message: Attn Devs — new local config defaults available.
~/code/sharedproject 💬
familiar acknowledge

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

```bash
familiar status
```

### Interact with Your Familiar

```bash
familiar feed      # Feed your familiar
familiar play      # Play with your familiar
familiar rest      # Let your familiar rest
familiar message "ship is red"  # Set a message
familiar acknowledge  # Acknowledge your familiar
```

### Prompt Integration

Add to your shell prompt (e.g., in `~/.bashrc` or `~/.zshrc`):

```bash
export PS1='$(familiar health) $ '
```

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
