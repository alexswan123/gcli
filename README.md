# gcli - Gmail and Google Calendar CLI

A command-line tool for managing Gmail and Google Calendar with support for multiple accounts.

## Features

- **Multi-account support**: Manage multiple Google accounts (e.g., work, personal)
- **Gmail operations**: Read, draft, send, and schedule emails
- **Calendar operations**: List, create, update, and delete events
- **Query flexibility**: Access one account or all accounts at once
- **JSON output**: Get structured output for scripting

## Installation

### Prerequisites

- Go 1.21 or later
- Google Cloud project with Gmail and Calendar APIs enabled

### Build from source

```bash
# Clone the repository
git clone https://github.com/alexandraswan/gcli.git
cd gcli

# Build the binary
make build

# Install to /usr/local/bin (requires sudo)
make install

# Or create a symlink instead
make link
```

## Google Cloud Setup

Before using gcli, you need to set up OAuth credentials:

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select an existing one
3. Enable the **Gmail API** and **Google Calendar API**
4. Go to "APIs & Services" > "Credentials"
5. Click "Create Credentials" > "OAuth client ID"
6. Select "Desktop app" as the application type
7. Add `http://localhost:8085/callback` as an authorized redirect URI
8. Save your Client ID and Client Secret

## Quick Start

### Add an account

```bash
# Add your first account
gcli auth add personal --client-id YOUR_CLIENT_ID --client-secret YOUR_CLIENT_SECRET

# Add another account
gcli auth add work --client-id WORK_CLIENT_ID --client-secret WORK_CLIENT_SECRET
```

The browser will open for OAuth authentication.

### Read emails

```bash
# Read from default account
gcli mail read

# Read from specific account
gcli mail read -a work

# Read from all accounts
gcli mail read --all

# Filter emails
gcli mail read -q "is:unread from:important@example.com"
```

### Send emails

```bash
# Create a draft
gcli mail draft -t "user@example.com" -s "Hello" -b "Message body"

# Send immediately
gcli mail send-now -t "user@example.com" -s "Hello" -b "Message body"

# Schedule for later
gcli mail schedule -t "user@example.com" -s "Hello" -b "Message body" --at "2024-12-25T10:00:00"
```

### Manage calendar

```bash
# List upcoming events
gcli cal list

# List events from all accounts
gcli cal list --all

# Create an event
gcli cal add -s "Meeting" --start "2024-12-25T10:00" --end "2024-12-25T11:00" -l "Conference Room"

# Update an event
gcli cal update EVENT_ID -s "Updated Meeting"

# Delete an event
gcli cal delete EVENT_ID
```

## Commands Reference

### Authentication (`gcli auth`)

| Command | Description |
|---------|-------------|
| `auth add <name>` | Add and authenticate a new account |
| `auth list` | List all configured accounts |
| `auth remove <name>` | Remove an account |
| `auth default <name>` | Set the default account |
| `auth reauth <name>` | Re-authenticate an existing account |

### Mail (`gcli mail`)

| Command | Description |
|---------|-------------|
| `mail read` | List emails |
| `mail get <id>` | Get email details |
| `mail draft` | Create a draft |
| `mail send <draft-id>` | Send an existing draft |
| `mail send-now` | Compose and send immediately |
| `mail schedule` | Schedule an email for later |
| `mail scheduled list` | List scheduled emails |
| `mail scheduled send` | Send ready scheduled emails |
| `mail scheduled clear` | Clear scheduled emails |

### Calendar (`gcli cal`)

| Command | Description |
|---------|-------------|
| `cal list` | List events |
| `cal get <id>` | Get event details |
| `cal add` | Create an event |
| `cal update <id>` | Update an event |
| `cal delete <id>` | Delete an event |
| `cal calendars` | List available calendars |

### Configuration (`gcli config`)

| Command | Description |
|---------|-------------|
| `config show` | Show current configuration |
| `config set <key> <value>` | Set a configuration value |
| `config path` | Show configuration file path |

## Global Flags

| Flag | Description |
|------|-------------|
| `-j, --json` | Output in JSON format |
| `-a, --account <name>` | Use specific account |
| `--all` | Use all accounts |

## Configuration

Configuration is stored in `~/.config/google-cli/`:

```
~/.config/google-cli/
├── config.json        # Account configurations
├── tokens/            # OAuth tokens per account
│   ├── personal.json
│   └── work.json
└── scheduled.json     # Scheduled emails
```

### Setting calendar ID

By default, gcli uses the primary calendar. To use a different calendar:

```bash
gcli config set work.calendar-id "work@company.com"
```

## Examples

### Read unread emails from all accounts

```bash
gcli mail read --all -q "is:unread" -n 50
```

### Send a scheduled email

```bash
# Schedule
gcli mail schedule -t "boss@company.com" -s "Weekly Report" -b "Here's my report..." --at "2024-12-25T09:00:00" -a work

# Send when ready (run via cron or manually)
gcli mail scheduled send
```

### List today's events

```bash
gcli cal list --from $(date +%Y-%m-%d) --to $(date +%Y-%m-%d) --all
```

### Create a meeting with attendees

```bash
gcli cal add -s "Team Standup" \
  --start "2024-12-25T10:00" \
  --end "2024-12-25T10:30" \
  -l "Zoom" \
  --attendees "alice@example.com,bob@example.com"
```

### Get JSON output for scripting

```bash
gcli mail read --json | jq '.[] | {from, subject}'
```

## Troubleshooting

### "No accounts configured"

Run `gcli auth add <name>` to add an account.

### "Authentication failed"

1. Verify your Client ID and Client Secret are correct
2. Ensure the redirect URI `http://localhost:8085/callback` is configured in Google Cloud Console
3. Try re-authenticating with `gcli auth reauth <name>`

### "Token expired"

Tokens are automatically refreshed. If issues persist, re-authenticate:

```bash
gcli auth reauth <account-name>
```

## License

MIT License
