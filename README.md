# GPT-3.5 Turbo Telegram Bot

This repository contains the source code for a Telegram bot that utilizes OpenAI's GPT-3.5 Turbo to assist users in answering questions and solving tasks.

## Prerequisites

To set up and run this bot, you'll need:

- Go (version 1.18 or higher)
- A Telegram bot token
- An OpenAI API key

## Installation

1. Clone this repository
2. Build the project:
```
go build -o gptbot
```

3. Rename a `bot.conf.sample` file to `bot.conf` in the project root directory and add your Telegram bot token, OpenAI API key, and other configuration values:
```
telegram_token = YOUR_TELEGRAM_BOT_TOKEN
gpt_token = YOUR_OPENAI_API_KEY
timeout_value = 60
max_messages = 10
admin_id = YOUR_TELEGRAM_ADMIN_USER_ID (optional)
```

Make sure to replace YOUR_TELEGRAM_BOT_TOKEN, YOUR_OPENAI_API_KEY, and other placeholders with your actual values.

## Running the bot
After building the project and creating the bot.conf file, run the bot:
```
./gptbot
```

The bot should now be running, and you can interact with it on Telegram. Send the `/start` command to begin using the bot.

Bot Commands
* /start - Sends a welcome message and describes the bot's purpose
* /help - Shows a list of available commands and their descriptions
* /clear - Clears the chat history for the current chat
* /history - Shows the current chat history in a formatted output
Contributing
Contributions are welcome! Please feel free to submit issues or pull requests for enhancements or bug fixes.
