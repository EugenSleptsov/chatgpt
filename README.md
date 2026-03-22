# OpenAI GPT model powered Telegram Bot

A Telegram bot powered by OpenAI GPT models. Supports private and group chats, multi-session conversations, image generation, voice transcription, auto-reply with configurable personas, and more.

## Prerequisites

- Go 1.26+
- A Telegram bot token
- An OpenAI API key

## Installation

1. Clone this repository
2. Build the project:
```bash
go build -o gptbot
```
3. Copy `config/bot.yaml.sample` to `config/bot.yaml` and fill in your tokens and settings:
```yaml
telegram_token: YOUR_TG_TOKEN
gpt_token: YOUR_OPENAI_TOKEN
timeout_value: 60
max_messages: 20
admin_id: 0
```

See [`config/bot.yaml.sample`](config/bot.yaml.sample) for the full list of options.

## Configuration

| Parameter | Description | Default                          |
|-----------|-------------|----------------------------------|
| `telegram_token` | Telegram Bot API token | *required*                       |
| `gpt_token` | OpenAI API key | *required*                       |
| `timeout_value` | Long-polling timeout (seconds) | `1`                              |
| `max_messages` | Max conversation messages kept in context | `20`                             |
| `admin_id` | Telegram user ID of the bot admin (`0` = disabled) | `0`                              |
| `ignore_report_ids` | User IDs excluded from admin reports | `[]`                             |
| `authorized_user_ids` | Authorized user IDs (empty = public bot) | `[]`                             |
| `command_menu` | Custom command menu (empty = default) | `[]`                             |
| `summarize_prompt` | System prompt for the `/summarize` command | *(built-in)*                     |
| `default_system_prompt` | Default system prompt for new sessions | `"You are a helpful assistant."` |
| `default_autoreply_persona` | Default role/persona for auto-reply decision (overridable per-chat via `/autorole`) | *(built-in)*                     |
| `telegram_token_log_bot` | Separate bot token for admin logging | `""`                             |
| `storage_type` | Storage backend: `file`, `sqlite`, or `memory` | `"file"`                         |
| `storage_dsn` | DSN for sqlite (path to db file) | `""`                             |
| `data_dir` | Directory for persistent data | `"_var/data"`                    |
| `log_dir` | Directory for log files | `"_var/log"`                     |

## Running the bot

```bash
./gptbot
```

The bot starts polling for Telegram updates. Send `/start` to begin.

In group chats the bot responds when mentioned via `@BotName`, replied to, or called by name (e.g. "бот"). With auto-reply enabled, the bot can also proactively join conversations.

## Bot Commands

### General
| Command | Description |
|---------|-------------|
| `/start` | Sends a welcome message |
| `/help` | Shows available commands |
| `/clear` | Clears conversation history for the current session |
| `/history [page]` | Shows conversation history (paginated) |
| `/rollback [n]` | Removes last *n* messages from history (default 1) |
| `/model [id]` | Shows or switches the AI model (`basic` / `advanced`) |
| `/system [text]` | Shows or sets the system prompt for the current session |
| `/markdown [on\|off]` | Toggles Markdown formatting in responses |
| `/memory` | Shows the bot's long-term memory for this chat. `/memory clear` to wipe |

### Sessions
| Command | Description |
|---------|-------------|
| `/list` | Lists all sessions |
| `/current` | Shows the active session |
| `/new <topic>` | Creates a new session and switches to it |
| `/use <id>` | Switches to a session by ID |
| `/update <id> <topic>` | Renames a session |
| `/remove <id>` | Deletes a session (cannot delete the last one) |

### GPT Text Tools
| Command | Description |
|---------|-------------|
| `/translate [lang] <text>` | Translates text to the specified language (default: English) |
| `/tech_translate <text>` | Translates text to technical English |
| `/enhance <text>` | Enhances text with more detail |
| `/grammar <text>` | Corrects grammar |
| `/summarize [n]` | Summarizes the last *n* chat messages (default 50, max 500) |
| `/summarize_prompt <text>` | Sets the system prompt for `/summarize` |
| `/analyze <n> <prompt>` | Analyzes last *n* messages using a custom prompt |

### Image
| Command | Description |
|---------|-------------|
| `/imagine <text>` | Generates a 1024×1024 image from a text description |

### Group Auto-Reply
| Command | Description |
|---------|-------------|
| `/autoreply` | Toggles proactive auto-reply mode in group chats |
| `/autorole [text]` | Shows or sets the bot's persona for auto-reply decisions. `/autorole reset` restores the default |

The auto-reply system uses a two-part prompt:
1. **Persona** — a configurable role describing who the bot "is" (e.g. teacher, listener, opinionated friend). Set globally via `default_autoreply_persona` in config, or per-chat via `/autorole`.
2. **Decision** — a fixed YES/NO instruction that determines whether the bot should respond based on conversation context.

### Admin
| Command | Description |
|---------|-------------|
| `/reload` | Reloads the configuration file |
| `/adduser <userId>` | Adds a user to the authorized list |
| `/removeuser <userId>` | Removes a user from the authorized list |

## Architecture

```
pipeline/          Transport-agnostic types (RequestContext, FileResolver)
  decoder/         Picks the right executor for each update
  executor/        Handles text, voice, image, sticker, command flows
  sender/          Delivers responses to Telegram

application/
  commands/        Slash-command implementations
  service/         Core services (GPT, history, memory, auth, config, notifier)

domain/
  ai/              AI client interface & tier definitions
  chat/            Chat & session domain models, storage interface

integration/ai/    OpenAI client implementation
infrastructure/    Storage backends (file, sqlite, memory), logger, utilities
api/telegram/      Telegram Bot API transport layer
app/               Wiring, worker pool, graceful shutdown
config/            YAML configuration
```

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## Special Thanks

Huge thanks to JetBrains for support, which greatly contributed to the development of this project.
https://www.jetbrains.com/community/opensource
