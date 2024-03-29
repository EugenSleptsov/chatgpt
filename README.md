# OpenAI GPT model powered Telegram Bot

This repository contains the source code for a Telegram bot that utilizes OpenAI's GPT models to assist users in answering questions and solving tasks.

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
ignore_report_ids = COMMA_SEPARATED_USER_IDS_TO_IGNORE (optional)
authorized_user_ids = COMMA_SEPARATED_USER_IDS (optional)
command_menu = COMMAND_LIST (optional)
```

Replace YOUR_TELEGRAM_BOT_TOKEN with your actual Telegram bot token and YOUR_OPENAI_API_KEY with your OpenAI API key.

Additionally, you can set the timeout_value, max_messages, admin_id, ignore_report_ids, and authorized_users to customize the bot's behavior. The admin_id, ignore_report_ids, and authorized_users are optional and can be left empty if not needed. If authorized_users is left empty, the bot will be available for public use.

You can also customize the command_menu to change the list of available commands. If command_menu is left empty, the default list of commands will be used.

## Running the bot
After building the project and creating the bot.conf file, run the bot:
```
./gptbot
```

The bot should now be running, and you can interact with it on Telegram. Send the `/start` command to begin using the bot. You can also add the bot to a group chat and use it there, but in such case you'll need to mention his name via @BotName or reply to any of his messages.

## Bot Commands
* /start - Sends a welcome message and describes the bot's purpose
* /help - Shows a list of available commands and their descriptions
* /history - Shows the current chat history in a formatted output
* /clear - Clears the chat history for the current chat
* /rollback `num` - Rolls back the chat history by `num` messages
* /translate `text` - Translates `text` from any language to English
* /grammar `text` - Checks the grammar of `text` and returns corrected text
* /enhance `text` - Enhances `text` by adding more details
* /imagine `text` - Generates an image based on `text`
* /temperature `value` - Sets the temperature value for the GPT-3.5 Turbo API
* /system `value` - Sets the system prompt for the GPT-3.5 Turbo API
* /model `value` - Sets openai model for API calls (gpt-3/gpt-4)
* /summarize `num` - Provides sarcastic summary for last `num` messages in group/private chat

## Admin Bot Commands
* /reload - Reloads config in case if you have changed parameters (for example, added new authorized users)
* /adduser userId - Adds a user to the authorized users list
* /removeuser userId - Removes a user from the authorized users list

## Contributing
Contributions are welcome! Please feel free to submit issues or pull requests for enhancements or bug fixes.

## Special Thanks
Huge thanks to JetBrains for support, which greatly contributed to the development of this project.
https://www.jetbrains.com/community/opensource
