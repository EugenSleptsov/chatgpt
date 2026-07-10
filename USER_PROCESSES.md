# Пользовательские процессы бота

```mermaid
flowchart TD
    U["Пользователь в Telegram"] --> UPD["Апдейт: сообщение / команда / кнопка / медиа"]
    REM["Планировщик каждые 30 сек"] --> DUE{"Есть due advisor reminders?"}
    DUE -- нет --> END0["Ничего"]
    DUE -- да --> SEND_REM["Отправить напоминание с кнопками: Готово / +1 час / Завтра"]

    UPD --> CTX["Собрать RequestContext"]
    CTX --> CHAT["Найти или создать Chat"]
    CHAT --> LOG["Записать сообщение в лог чата"]
    LOG --> PENDING{"Есть PendingInput?"}
    PENDING -- да, ответ на ForceReply --> ROUTE_PENDING["Превратить текст в команду"]
    PENDING -- нет --> AUTH
    ROUTE_PENDING --> AUTH["Проверка доступа"]

    AUTH --> PRIVATE_DENY{"Личка и пользователь не авторизован?"}
    PRIVATE_DENY -- да --> DENY["Ответ: нет доступа + уведомить админа"]
    PRIVATE_DENY -- нет --> GROUP_DENY{"Группа, команда, пользователь не авторизован?"}
    GROUP_DENY -- да --> END1["Игнор"]
    GROUP_DENY -- нет --> DECODER["Decoder выбирает executor"]

    DECODER --> IS_CMD{"Команда или callback-кнопка?"}
    IS_CMD -- да --> CMD["CommandExecutor"]
    IS_CMD -- нет --> IS_VOICE{"Голосовое?"}
    IS_VOICE -- да --> VOICE["VoiceExecutor"]
    IS_VOICE -- нет --> IS_PHOTO{"Фото?"}
    IS_PHOTO -- да --> PHOTO["ImageExecutor"]
    IS_PHOTO -- нет --> IS_STICKER{"Стикер?"}
    IS_STICKER -- да --> STICKER["StickerExecutor"]
    IS_STICKER -- нет --> TEXT["TextExecutor"]

    CMD --> CMD_KIND{"Какая команда/кнопка?"}

    CMD_KIND --> MENU["/menu или /help"]
    MENU --> MAIN_MENU["Главное меню"]
    MAIN_MENU --> SESS["Сессии"]
    MAIN_MENU --> ADV["Advisor заметки"]
    MAIN_MENU --> SETT["Настройки"]
    MAIN_MENU --> TOOLS["Инструменты"]
    MAIN_MENU --> INFO["Инфо"]

    SESS --> SESS_FLOW["Список / выбрать / новая / переименовать / удалить с подтверждением"]
    SESS_FLOW --> FORCE1{"Нужен текст?"}
    FORCE1 -- да --> FORCE_REPLY1["ForceReply: следующий ответ станет /new или /rename"]
    FORCE1 -- нет --> SEND

    ADV --> ADV_FLOW["Темы / записи / удалить запись / удалить тему / объединить темы"]
    ADV --> ADV_REM["Кнопки напоминаний: done / snooze"]
    ADV_REM --> ADV_DONE{"Готово или отложить?"}
    ADV_DONE -- готово --> ADV_DELETE["Удалить advisor entry"]
    ADV_DONE -- отложить --> ADV_SNOOZE["Поставить новое RemindAt"]
    ADV_FLOW --> SEND
    ADV_DELETE --> SEND
    ADV_SNOOZE --> SEND

    SETT --> SETT_FLOW["Модель / Markdown / подтверждение удаления / system prompt / summarize prompt / память"]
    SETT --> SETT_ADMIN["Админ: auto-reply / autorole / verbose"]
    SETT_FLOW --> FORCE2{"Нужен текст?"}
    FORCE2 -- да --> FORCE_REPLY2["ForceReply: следующий ответ станет /system, /summarize_prompt или /autorole"]
    FORCE2 -- нет --> SEND
    SETT_ADMIN --> SEND

    TOOLS --> TOOL_KIND{"Инструмент"}
    TOOL_KIND --> IMAGINE["/imagine: генерация картинки, cooldown 15 мин"]
    TOOL_KIND --> TRANSLATE["/translate"]
    TOOL_KIND --> TECH["/tech_translate"]
    TOOL_KIND --> ENHANCE["/enhance"]
    TOOL_KIND --> GRAMMAR["/grammar"]
    TOOL_KIND --> SUMM["/summarize: последние N сообщений из log"]
    TOOL_KIND --> ANALYZE["/analyze: последние N сообщений + промпт"]
    IMAGINE --> IMAGE_ONE_SHOT["GPTService → OneShotService.GenerateImage"]
    IMAGE_ONE_SHOT --> SEND
    TRANSLATE --> ONE_SHOT_GPT["GPTService → OneShotService.GPTCommand без истории"]
    TECH --> ONE_SHOT_GPT
    ENHANCE --> ONE_SHOT_GPT
    GRAMMAR --> ONE_SHOT_GPT
    SUMM --> ONE_SHOT_GPT
    ANALYZE --> ONE_SHOT_GPT
    ONE_SHOT_GPT --> SEND

    INFO --> USAGE["/usage: расходы и токены"]
    INFO --> CONTEXT["/context: оценка контекста"]
    INFO --> HISTORY["/history: история"]
    INFO --> HELP_LIST["/help list: список команд"]
    USAGE --> SEND
    CONTEXT --> SEND
    HISTORY --> SEND
    HELP_LIST --> SEND

    CMD_KIND --> SIMPLE["/start /clear /rollback /system /summarize_prompt /autorole"]
    SIMPLE --> SEND
    CMD_KIND --> ADMIN["Админ: /reload /adduser /removeuser"]
    ADMIN --> SEND

    VOICE --> TRANSCRIBE["Скачать файл и расшифровать аудио"]
    TRANSCRIBE --> FWD{"Forwarded voice?"}
    FWD -- да --> SEND_TRANSCRIPT["Отправить только расшифровку"]
    FWD -- нет --> VOICE_TEXT["Отправить расшифровку и обработать как текст"]
    VOICE_TEXT --> TEXT
    SEND_TRANSCRIPT --> SEND

    PHOTO --> GROUP_PHOTO{"Фото в группе?"}
    GROUP_PHOTO -- нет --> VISION["GPT Vision: caption или 'опишите изображение'"]
    GROUP_PHOTO -- да --> PHOTO_ADDR{"Бот упомянут или фото reply на бота?"}
    PHOTO_ADDR -- нет --> LOG_PHOTO["Только залогировать фото"]
    PHOTO_ADDR -- да --> VISION
    VISION --> SEND
    LOG_PHOTO --> END2["Ничего"]

    STICKER --> STICKER_GROUP{"Стикер в группе?"}
    STICKER_GROUP -- да --> LOG_STICKER["Записать стикер в контекст группы"]
    STICKER_GROUP -- нет --> END3["Ничего"]
    LOG_STICKER --> END3

    TEXT --> CHAT_TYPE{"Личка или группа?"}
    CHAT_TYPE -- личка --> PRIV_TEXT["Добавить user message в историю"]
    PRIV_TEXT --> COMPLETE["GPTService.Complete"]

    CHAT_TYPE -- группа --> GROUP_TEXT["Записать сообщение в историю группы"]
    GROUP_TEXT --> EDITED{"Сообщение отредактировано?"}
    EDITED -- да --> END4["Только обновили контекст"]
    EDITED -- нет --> ADDR{"Бот упомянут / reply / слово 'бот'?"}
    ADDR -- да --> COMPLETE
    ADDR -- нет --> AUTO{"GroupAutoReply включен?"}
    AUTO -- нет --> END5["Не отвечать"]
    AUTO -- да --> AUTH_AUTO{"Авторизованный отправитель?"}
    AUTH_AUTO -- нет --> END6["Не отвечать"]
    AUTH_AUTO -- да --> DECIDE["GPTService → OneShotService.ShouldAutoReply: YES/NO"]
    DECIDE --> SHOULD{"YES?"}
    SHOULD -- нет --> END7["Не отвечать"]
    SHOULD -- да --> COMPLETE

    COMPLETE --> COST{"Дневной лимит расходов превышен?"}
    COST -- да --> LIMIT_MSG["Ответ о превышении лимита"]
    COST -- нет --> COMPACT{"Контекст близко к лимиту?"}
    COMPACT -- да --> AUTO_COMPACT["Сжать старую историю"]
    COMPACT -- нет --> BUILD
    AUTO_COMPACT --> BUILD["Собрать prompt: system + memory + advisor topics + дата + style"]

    BUILD --> GPT["OpenAI CallGPT с history и tools"]
    GPT --> RUNNER["ToolRunner.Run"]
    RUNNER --> BUILTIN["Учесть server-side tools: web_search / image_generation"]
    BUILTIN --> TOOLS_LOOP{"Есть client-side function_call?"}
    TOOLS_LOOP -- нет --> FINAL_TEXT["Взять финальный ответ"]
    TOOLS_LOOP -- да --> TOOL_SELECT{"Какой function tool?"}

    TOOL_SELECT --> VOICE_TOOL["generate_voice"]
    TOOL_SELECT --> MEM_TOOL["update_memory"]
    TOOL_SELECT --> SAVE_NOTE["save_note"]
    TOOL_SELECT --> READ_NOTE["read_notes"]
    TOOL_SELECT --> SET_REM["set_reminder"]

    VOICE_TOOL --> CONTINUE["ContinueWithToolOutputs"]
    MEM_TOOL --> CONTINUE
    SAVE_NOTE --> CONTINUE
    READ_NOTE --> CONTINUE
    SET_REM --> CONTINUE
    CONTINUE --> RUNNER

    FINAL_TEXT --> SAVE_HISTORY["Сохранить assistant response в историю"]
    LIMIT_MSG --> SAVE_HISTORY
    SAVE_HISTORY --> VOICE_AUDIO{"Вход был voice и аудио не создано?"}
    VOICE_AUDIO -- да --> TTS["Сгенерировать голосовой ответ"]
    VOICE_AUDIO -- нет --> SEND
    TTS --> SEND

    SEND["Отправить ответ в Telegram"]
    SEND --> HUB{"Ответ с inline-кнопками?"}
    HUB -- да --> TRACK_HUB["Запомнить hub message; старый hub удалить best-effort"]
    HUB -- нет --> SAVE_CHAT["MarkDirty + Save storage"]
    TRACK_HUB --> SAVE_CHAT
```

## Общая верхнеуровневая схема

```mermaid
flowchart LR
    USER["Пользователь"] --> TG["Telegram"]
    TG --> APP["Bot app"]
    APP --> AUTH["Доступ"]
    AUTH --> ROUTER["Маршрутизация"]

    ROUTER --> COMMANDS["Команды и кнопки"]
    ROUTER --> CHAT["Обычный диалог"]
    ROUTER --> MEDIA["Медиа"]

    COMMANDS --> SERVICES["Сервисы бота"]
    CHAT --> GPT_SERVICE["GPTService facade"]
    MEDIA --> GPT_SERVICE
    SERVICES --> GPT_SERVICE

    GPT_SERVICE --> ONE_SHOT["OneShotService: команды / vision / images / auto-reply"]
    GPT_SERVICE --> CHAT_FLOW["Complete: история / compact / usage"]
    CHAT_FLOW --> TOOL_RUNNER["ToolRunner: tools / images / voice / memory / advisor"]
    ONE_SHOT --> OPENAI["OpenAI Responses / Images / Audio"]
    TOOL_RUNNER --> OPENAI
    CHAT_FLOW --> OPENAI

    OPENAI --> RESPONSE["Ответ"]
    SERVICES --> RESPONSE
    RESPONSE --> TG

    APP <--> STORAGE["Storage file/memory: chats, sessions, memory, advisor"]
    SCHED["Reminder scheduler"] --> APP
```

## Запуск и конфиг

```mermaid
flowchart TD
    CFG["config/bot.yaml"] --> READ["ReadConfig + ApplyDefaults"]
    READ --> TIMEOUT["timeout_value (default 1)"]
    READ --> STORAGE_TYPE["storage_type: file или memory"]
    READ --> GPT_CFG["gpt_token + cost_limit_usd"]

    TIMEOUT --> POLL["Telegram GetUpdateChannel(timeout_value)"]
    STORAGE_TYPE --> STORAGE["NewStorage(file/memory)"]
    GPT_CFG --> GPT_SERVICE["NewGPTService"]

    GPT_SERVICE --> COMPACT["CompactService"]
    GPT_SERVICE --> ONE_SHOT["OneShotService"]
    GPT_SERVICE --> TOOL_RUNNER["ToolRunner"]
```

## Вход, доступ и маршрутизация

```mermaid
flowchart TD
    UPD["Telegram update"] --> CTX["RequestContext"]
    CTX --> CHAT["GetOrCreateChat"]
    CHAT --> LOG["Лог сообщения"]
    LOG --> PENDING{"Ожидается ForceReply?"}

    PENDING -- да --> AS_COMMAND["Текст становится аргументами команды"]
    PENDING -- нет --> AUTH["Проверка доступа"]
    AS_COMMAND --> AUTH

    AUTH --> DENY{"Нет доступа?"}
    DENY -- да --> STOP["Игнор или сообщение 'нет доступа'"]
    DENY -- нет --> DECODER["Decoder"]

    DECODER --> CMD["CommandExecutor"]
    DECODER --> VOICE["VoiceExecutor"]
    DECODER --> PHOTO["ImageExecutor"]
    DECODER --> STICKER["StickerExecutor"]
    DECODER --> TEXT["TextExecutor"]
```

## Обычный GPT-диалог

```mermaid
flowchart TD
    TEXT["Текст пользователя"] --> SCOPE{"Личка или группа?"}
    SCOPE -- личка --> HIST["Добавить сообщение в историю"]
    SCOPE -- группа --> GROUP_RULES["Проверить обращение к боту или auto-reply"]

    GROUP_RULES --> REPLY{"Нужно отвечать?"}
    REPLY -- нет --> STOP["Не отвечать"]
    REPLY -- да --> HIST

    HIST --> COMPLETE["GPTService.Complete"]
    COMPLETE --> LIMIT{"Дневной лимит расходов?"}
    LIMIT -- превышен --> LIMIT_MSG["Ответ о лимите"]
    LIMIT -- ok --> COMPACT{"Контекст близко к лимиту?"}
    COMPACT -- да --> SUMMARY["Сжать старую историю"]
    COMPACT -- нет --> PROMPT
    SUMMARY --> PROMPT["Собрать prompt: system + memory + advisor topics + дата"]

    PROMPT --> GPT["OpenAI CallGPT с tools"]
    GPT --> RUNNER["ToolRunner.Run"]
    RUNNER --> TOOL_LOOP{"Есть client-side function_call?"}
    TOOL_LOOP -- да --> TOOLS["Выполнить tool + ContinueWithToolOutputs"]
    TOOLS --> RUNNER
    TOOL_LOOP -- нет --> ANSWER["Финальный ответ"]

    LIMIT_MSG --> SAVE["Сохранить assistant response"]
    ANSWER --> SAVE
    SAVE --> SEND["Отправить в Telegram"]
```

## Команды и кнопки

```mermaid
flowchart TD
    CMD["/command или callback"] --> REG["Command registry"]
    REG --> KIND{"Раздел"}

    KIND --> MENU["Меню"]
    KIND --> SESS["Сессии"]
    KIND --> ADVISOR["Advisor"]
    KIND --> SETTINGS["Настройки"]
    KIND --> TOOLS["Инструменты"]
    KIND --> INFO["Инфо"]
    KIND --> ADMIN["Админ"]

    MENU --> BUTTONS["Показать inline-кнопки"]
    SESS --> SESS_OPS["Выбрать / создать / переименовать / удалить"]
    ADVISOR --> ADV_OPS["Просмотр / удаление / объединение / reminder actions"]
    SETTINGS --> SET_OPS["Модель / Markdown / prompts / memory / auto-reply"]
    TOOLS --> TOOL_OPS["translate / enhance / grammar / summarize / analyze / imagine"]
    INFO --> INFO_OPS["usage / context / history / help list"]
    ADMIN --> ADMIN_OPS["reload / adduser / removeuser"]

    SESS_OPS --> NEED_INPUT{"Нужен текст?"}
    SET_OPS --> NEED_INPUT
    TOOL_OPS --> NEED_INPUT

    NEED_INPUT -- да --> FORCE["ForceReply + PendingInput"]
    NEED_INPUT -- нет --> SEND["Ответ пользователю"]
    BUTTONS --> SEND
    ADV_OPS --> SEND
    INFO_OPS --> SEND
    ADMIN_OPS --> SEND
```

## Advisor-заметки и напоминания

```mermaid
flowchart TD
    USER["Пользователь"] --> ASK{"Что делает пользователь?"}

    ASK -- "просит записать" --> SAVE_NOTE["ToolRunner: save_note"]
    ASK -- "спрашивает по заметкам" --> READ_NOTES["ToolRunner: read_notes"]
    ASK -- "меняет напоминание" --> SET_REM["ToolRunner: set_reminder"]
    ASK -- "открывает меню заметок" --> HUB["/advisor hub"]

    SAVE_NOTE --> TOPIC["Создать или найти тему"]
    TOPIC --> ENTRY["Добавить запись"]

    READ_NOTES --> TOPIC_DATA["Вернуть записи темы модели"]
    SET_REM --> FIND_ENTRY["Найти entry по topic + id"]
    FIND_ENTRY --> REMIND_AT["Поставить или очистить RemindAt"]

    HUB --> MANAGE["Просмотр / удалить запись / удалить тему / объединить темы"]

    ENTRY --> STORAGE["Chat.AdvisorTopics"]
    REMIND_AT --> STORAGE
    MANAGE --> STORAGE

    SCHED["Reminder scheduler"] --> DUE{"Есть просроченные RemindAt?"}
    DUE -- да --> SEND_REM["Отправить reminder с кнопками"]
    SEND_REM --> ACTION{"Кнопка"}
    ACTION -- "Готово" --> DELETE["Удалить запись"]
    ACTION -- "Отложить" --> SNOOZE["Новый RemindAt"]
```

## Групповой чат

```mermaid
flowchart TD
    MSG["Сообщение в группе"] --> LOG["Записать в историю группы"]
    LOG --> EDITED{"Edited message?"}
    EDITED -- да --> STOP1["Только обновить контекст"]
    EDITED -- нет --> ADDRESSED{"Бот упомянут / reply / слово 'бот'?"}

    ADDRESSED -- да --> COMPLETE["GPTService.Complete"]
    ADDRESSED -- нет --> AUTO{"GroupAutoReply включен?"}
    AUTO -- нет --> STOP2["Не отвечать"]
    AUTO -- да --> AUTH{"Отправитель авторизован?"}
    AUTH -- нет --> STOP3["Не отвечать"]
    AUTH -- да --> DECIDE["GPTService → OneShotService.ShouldAutoReply"]
    DECIDE --> YESNO{"YES?"}
    YESNO -- нет --> STOP4["Не отвечать"]
    YESNO -- да --> COMPLETE

    COMPLETE --> REPLY["Ответ в группу"]
```

## Медиа-сценарии

```mermaid
flowchart TD
    MEDIA["Медиа от пользователя"] --> TYPE{"Тип"}

    TYPE --> VOICE["Голосовое"]
    VOICE --> TRANSCRIBE["TranscribeAudio"]
    TRANSCRIBE --> FWD{"Forwarded?"}
    FWD -- да --> ONLY_TEXT["Отправить только расшифровку"]
    FWD -- нет --> AS_TEXT["Расшифровка идет в TextExecutor"]

    TYPE --> PHOTO["Фото"]
    PHOTO --> GROUP_PHOTO{"Фото в группе?"}
    GROUP_PHOTO -- нет --> VISION["GPT Vision"]
    GROUP_PHOTO -- да --> ADDR{"Бот упомянут или reply?"}
    ADDR -- нет --> PHOTO_LOG["Только лог"]
    ADDR -- да --> VISION

    TYPE --> STICKER["Стикер"]
    STICKER --> GROUP_STICKER{"В группе?"}
    GROUP_STICKER -- да --> STICKER_LOG["Записать в контекст"]
    GROUP_STICKER -- нет --> IGNORE["Игнор"]

    VISION --> SEND["Ответ"]
    AS_TEXT --> SEND
    ONLY_TEXT --> SEND
```
