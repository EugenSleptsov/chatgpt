package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// builtinTools are executed server-side by OpenAI: web_search returns text,
// image_generation returns base64 PNGs in the response output.
var builtinTools = []ai.Tool{
	{Type: "web_search"},
	{Type: "image_generation"},
}

// functionTools are executed client-side by executeSingleToolCall.
var functionTools = []ai.Tool{
	{
		Type:        "function",
		Name:        "generate_voice",
		Description: "Convert text to a voice/audio message. Call when the user asks to record voice, speak out loud, or create audio.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"text": {Type: "string", Description: "The text to convert to speech"},
			},
			Required: []string{"text"},
		},
	},
	{
		Type:        "function",
		Name:        "update_memory",
		Description: "Save or update a fact about the user/chat for future conversations. Call when you learn something worth remembering (name, preferences, context) or when the user explicitly asks to remember something. Each call adds one fact. Existing memory is shown in the system prompt. For concrete notes/todo/list items ('запиши', 'напомни', shopping items, tasks) use save_note instead.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"fact": {Type: "string", Description: "A single fact to remember, e.g. 'User prefers dark mode' or 'User's name is Alex'"},
			},
			Required: []string{"fact"},
		},
	},
	{
		Type:        "function",
		Name:        "save_note",
		Description: "Save a concrete note into a topical list (advisor). Call when the user asks to write something down, note it, or keep it for later ('запиши', 'запомни этот пункт', 'напомни мне про...'). Pick an existing topic from the system prompt when one fits, otherwise invent a short topic name in the user's language (e.g. 'Налоги', 'Покупки в ИКЕА'). One call per note. When the user mentions a date or time ('завтра в 15', 'в пятницу'), resolve it against the current date from the system prompt and pass remind_at. Unlike update_memory (facts about the user), notes are list items the user will review and delete later.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"topic":     {Type: "string", Description: "Topic (list) name to file the note under; reuse an existing topic when it fits"},
				"note":      {Type: "string", Description: "The note text, one self-contained item"},
				"remind_at": {Type: "string", Description: "Optional reminder time, 'YYYY-MM-DD HH:MM' or 'YYYY-MM-DD' (defaults to morning). Omit for a plain note."},
			},
			Required: []string{"topic", "note"},
		},
	},
	{
		Type:        "function",
		Name:        "read_notes",
		Description: "Read all saved advisor notes of one topic, including entry IDs and reminder times. Call when the user asks what is saved under a topic ('что у меня по налогам?') or before changing a reminder on an existing note. Topic names are listed in the system prompt.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"topic": {Type: "string", Description: "Topic name to read"},
			},
			Required: []string{"topic"},
		},
	},
	{
		Type:        "function",
		Name:        "set_reminder",
		Description: "Set, move or clear the reminder of an existing advisor note ('передвинь на 16:00', 'напомни про эту заметку завтра', 'убери напоминание'). Call read_notes first to find the entry ID. Resolve relative dates against the current date from the system prompt.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"topic":     {Type: "string", Description: "Topic name the note lives in"},
				"entry_id":  {Type: "string", Description: "Entry ID from read_notes (the #N number)"},
				"remind_at": {Type: "string", Description: "New reminder time, 'YYYY-MM-DD HH:MM' or 'YYYY-MM-DD'. Empty string clears the reminder."},
			},
			Required: []string{"topic", "entry_id"},
		},
	},
}

// chatTools is the single tool set sent on every chat completion.
var chatTools = concatTools(builtinTools, functionTools)

func concatTools(a, b []ai.Tool) []ai.Tool {
	r := make([]ai.Tool, 0, len(a)+len(b))
	return append(append(r, a...), b...)
}

func toolNamesFromTools(tools []ai.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t.Name != "" {
			names = append(names, t.Name)
		} else {
			names = append(names, t.Type)
		}
	}
	return names
}

// executeSingleToolCall runs one tool call and returns the JSON output string.
func (s *GPTService) executeSingleToolCall(tc ai.ToolCall, result *ChatResult, chat *chatdomain.Chat) string {
	log.Printf("[ToolCall] %s(%v)", tc.Name, tc.Args)
	switch tc.Name {
	case "generate_voice":
		return s.executeVoiceToolCall(tc, result, chat.ChatID)
	case "update_memory":
		return s.executeUpdateMemory(tc, chat)
	case "save_note":
		return s.executeSaveNote(tc, chat)
	case "read_notes":
		return s.executeReadNotes(tc, chat)
	case "set_reminder":
		return s.executeSetReminder(tc, chat)
	default:
		log.Printf("[ToolCall] unknown tool: %s", tc.Name)
		return marshalToolResult(toolResult{Status: "error", Error: "unknown tool: " + tc.Name})
	}
}

func (s *GPTService) executeVoiceToolCall(tc ai.ToolCall, result *ChatResult, chatID int64) string {
	text := tc.Args["text"]
	if text == "" {
		text = result.Text
	}
	if text == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "no text available for voice synthesis"})
	}
	done := StartProgress(s.Progress, chatID, "🎙 Идет генерация аудио…")
	defer done()
	audio, err := s.GptClient.GenerateVoice(text, ai.VoiceModelHD, ai.VoiceOnyx)
	if err != nil {
		log.Printf("[ToolCall] generate_voice error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	result.Audio = audio
	result.AudioText = text
	return marshalToolResult(toolResult{Status: "success", Text: text})
}

func (s *GPTService) executeUpdateMemory(tc ai.ToolCall, chat *chatdomain.Chat) string {
	fact := strings.TrimSpace(tc.Args["fact"])
	if fact == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "empty fact"})
	}
	AddMemory(chat, fact)
	return marshalToolResult(toolResult{Status: "success", Text: "Fact saved"})
}

func (s *GPTService) executeSaveNote(tc ai.ToolCall, chat *chatdomain.Chat) string {
	remindAt, err := ParseRemindAt(tc.Args["remind_at"])
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	topic, err := AddAdvisorNote(chat, tc.Args["topic"], tc.Args["note"], remindAt)
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	text := fmt.Sprintf("Note saved to topic %q (%d entries)", topic.Name, len(topic.Entries))
	if remindAt != nil {
		text += ", reminder at " + remindAt.Format("2006-01-02 15:04")
	}
	return marshalToolResult(toolResult{Status: "success", Text: text})
}

func (s *GPTService) executeSetReminder(tc ai.ToolCall, chat *chatdomain.Chat) string {
	topic := chat.FindAdvisorTopicByName(tc.Args["topic"])
	if topic == nil {
		return marshalToolResult(toolResult{Status: "error", Error: "topic not found: " + tc.Args["topic"]})
	}
	entryID, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(tc.Args["entry_id"], "#")))
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: "invalid entry_id: " + tc.Args["entry_id"]})
	}
	remindAt, err := ParseRemindAt(tc.Args["remind_at"])
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	if !chat.SetAdvisorReminder(topic.ID, entryID, remindAt) {
		return marshalToolResult(toolResult{Status: "error", Error: fmt.Sprintf("entry #%d not found in topic %q", entryID, topic.Name)})
	}
	if remindAt == nil {
		return marshalToolResult(toolResult{Status: "success", Text: fmt.Sprintf("Reminder cleared on entry #%d", entryID)})
	}
	return marshalToolResult(toolResult{Status: "success", Text: fmt.Sprintf("Reminder on entry #%d set to %s", entryID, remindAt.Format("2006-01-02 15:04"))})
}

func (s *GPTService) executeReadNotes(tc ai.ToolCall, chat *chatdomain.Chat) string {
	topic := chat.FindAdvisorTopicByName(tc.Args["topic"])
	if topic == nil {
		return marshalToolResult(toolResult{Status: "error", Error: "topic not found: " + tc.Args["topic"]})
	}
	return marshalToolResult(toolResult{Status: "success", Text: AdvisorNotesForTool(topic)})
}

// toolResult is the JSON structure returned by tool call handlers.
type toolResult struct {
	Status string `json:"status"`
	Text   string `json:"text,omitempty"`
	Error  string `json:"error,omitempty"`
}

func marshalToolResult(r toolResult) string {
	data, _ := json.Marshal(r)
	return string(data)
}
