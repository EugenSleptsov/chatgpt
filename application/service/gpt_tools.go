package service

import (
	"GPTBot/domain/ai"
	chatdomain "GPTBot/domain/chat"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// builtinTools are executed server-side by OpenAI: web_search returns text,
// image_generation returns base64 PNGs in the response output.
var builtinTools = []ai.Tool{
	{Type: "web_search"},
	{Type: "image_generation"},
}

// functionTools are executed client-side by ToolRunner.
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
		Name:        "save_note",
		Description: "Save a concrete note into a topical list (advisor). Call when the user asks to write something down, note it, or keep it for later ('запиши', 'запомни этот пункт', 'напомни мне про...'). Pick an existing topic from the system prompt when one fits, otherwise invent a short topic name in the user's language (e.g. 'Налоги', 'Покупки в ИКЕА'). One call per note. When the user mentions a date or time ('завтра в 15', 'в пятницу'), resolve it against the current date from the system prompt and pass remind_at. Notes are list items the user will review and delete later.",
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

// supermemoryTools are exposed only when the chat's Supermemory setting is on.
var supermemoryTools = []ai.Tool{
	{
		Type:        "function",
		Name:        "search_memory",
		Description: "Search the long-term memory (supermemory) of this chat by keyword: matches node hooks and summaries across all layers, including nodes not shown in the index. Call when the user refers to something from past conversations that is not in the current context ('мы это уже обсуждали', 'как я тогда говорил').",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"query": {Type: "string", Description: "Keyword or phrase to search for, in the language of the conversation"},
			},
			Required: []string{"query"},
		},
	},
	{
		Type:        "function",
		Name:        "read_memory",
		Description: "Read one supermemory node: its full summary, child nodes (descend deeper by calling read_memory again) and whether a raw transcript is available. Node IDs come from the supermemory index in the system prompt or from search_memory.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"node_id": {Type: "string", Description: "Node ID (the #N number)"},
			},
			Required: []string{"node_id"},
		},
	},
	{
		Type:        "function",
		Name:        "save_memory",
		Description: "Force-save the conversation into long-term memory right now: everything not yet covered by a memory node is summarized into a new node immediately, without waiting for the automatic threshold. Call when the user asks to remember or memorize something from the conversation ('запомни это', 'сохрани в память'), or when an important conclusion/decision was just reached and must not be lost.",
		Parameters: &ai.FunctionParameters{
			Type:       "object",
			Properties: map[string]ai.ParameterProperty{},
		},
	},
	{
		Type:        "function",
		Name:        "read_memory_source",
		Description: "Read the raw archived transcript behind a supermemory node, verbatim. Use after read_memory when the summary is not detailed enough (exact wording, numbers, code). Output is capped; prefer summaries when they suffice.",
		Parameters: &ai.FunctionParameters{
			Type: "object",
			Properties: map[string]ai.ParameterProperty{
				"node_id": {Type: "string", Description: "Node ID (the #N number)"},
			},
			Required: []string{"node_id"},
		},
	},
}

// toolsForChat assembles the tool set for one completion: the base tools plus
// the supermemory tools when the chat has the feature enabled.
func toolsForChat(c *chatdomain.Chat) []ai.Tool {
	tools := concatTools(builtinTools, functionTools)
	if c.Settings.Supermemory {
		tools = append(tools, supermemoryTools...)
	}
	return tools
}

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

type ToolRunner struct {
	Client    ai.Client
	CostFn    CostFunc
	ImageCost float64
	Progress  ProgressReporter
	Archive   chatdomain.Archive // raw transcript store for read_memory_source (may be nil)
	Compact   *CompactService    // snapshot machinery for save_memory (may be nil)
}

const maxToolIterations = 5

func (r ToolRunner) Run(response *ai.Response, model, instructions string, chat *chatdomain.Chat, tools []ai.Tool, initialPhase string) (*ChatResult, error) {
	result := &ChatResult{}
	tNames := toolNamesFromTools(tools)
	result.Usage.add(extractUsage(response, model, initialPhase, r.CostFn, tNames...))
	for i := 0; i < maxToolIterations; i++ {
		r.collectImages(response, result)
		calls := response.ToolCalls()
		r.announceTools(chat, response, calls)
		if len(calls) == 0 {
			result.Text = strings.TrimSpace(response.OutputText())
			return result, nil
		}
		if text := strings.TrimSpace(response.OutputText()); text != "" {
			result.Text = text
		}
		log.Printf("[ToolLoop] iteration %d: %d tool call(s)", i+1, len(calls))
		outputs := make([]ai.ToolCallOutput, 0, len(calls))
		for _, tc := range calls {
			output := r.executeSingleToolCall(tc, result, chat)
			outputs = append(outputs, ai.NewToolCallOutput(tc.ID, output))
		}
		var err error
		response, err = r.Client.ContinueWithToolOutputs(response.ID, outputs, model, instructions, tools...)
		if err != nil {
			log.Printf("[ToolLoop] error continuing response: %v", err)
			if result.Text == "" {
				result.Text = fallbackResponse
			}
			return result, err
		}
		calledNames := make([]string, 0, len(calls))
		for _, tc := range calls {
			calledNames = append(calledNames, tc.Name)
		}
		result.Usage.add(extractUsage(response, model, fmt.Sprintf("Continue (%s)", strings.Join(calledNames, ", ")), r.CostFn, tNames...))
	}
	log.Printf("[ToolLoop] max iterations (%d) reached", maxToolIterations)
	r.collectImages(response, result)
	r.announceTools(chat, response, nil)
	result.Text = strings.TrimSpace(response.OutputText())
	if result.Text == "" {
		result.Text = fallbackResponse
	}
	return result, nil
}

func (r ToolRunner) collectImages(response *ai.Response, result *ChatResult) {
	for _, imgData := range response.ImageResults() {
		result.Images = append(result.Images, ImageResult{Data: imgData})
		result.Usage.addFixedCost("DALL-E (image)", r.ImageCost)
	}
}

// announceTools reports tool invocations into the chat when verbose mode is on.
func (r ToolRunner) announceTools(chat *chatdomain.Chat, response *ai.Response, calls []ai.ToolCall) {
	if !chat.Settings.Verbose {
		return
	}
	for _, name := range response.BuiltinCalls() {
		Announce(r.Progress, chat.ChatID, "🔧 Вызван "+name)
	}
	for _, tc := range calls {
		Announce(r.Progress, chat.ChatID, "🔧 Вызван "+tc.Name)
	}
}

// executeSingleToolCall runs one tool call and returns the JSON output string.
func (r ToolRunner) executeSingleToolCall(tc ai.ToolCall, result *ChatResult, chat *chatdomain.Chat) string {
	log.Printf("[ToolCall] %s(%v)", tc.Name, tc.Args)
	switch tc.Name {
	case "generate_voice":
		return r.executeVoiceToolCall(tc, result, chat.ChatID)
	case "search_memory":
		return r.executeSearchMemory(tc, chat)
	case "read_memory":
		return r.executeReadMemory(tc, chat)
	case "read_memory_source":
		return r.executeReadMemorySource(tc, chat)
	case "save_memory":
		return r.executeSaveMemory(chat)
	case "save_note":
		return r.executeSaveNote(tc, chat)
	case "read_notes":
		return r.executeReadNotes(tc, chat)
	case "set_reminder":
		return r.executeSetReminder(tc, chat)
	default:
		log.Printf("[ToolCall] unknown tool: %s", tc.Name)
		return marshalToolResult(toolResult{Status: "error", Error: "unknown tool: " + tc.Name})
	}
}

func (r ToolRunner) executeVoiceToolCall(tc ai.ToolCall, result *ChatResult, chatID int64) string {
	text := tc.Args["text"]
	if text == "" {
		text = result.Text
	}
	if text == "" {
		return marshalToolResult(toolResult{Status: "error", Error: "no text available for voice synthesis"})
	}
	done := StartProgress(r.Progress, chatID, "🎙 Идет генерация аудио…")
	defer done()
	audio, err := r.Client.GenerateVoice(text, ai.VoiceModelHD, ai.VoiceOnyx)
	if err != nil {
		log.Printf("[ToolCall] generate_voice error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	result.Audio = audio
	result.AudioText = text
	return marshalToolResult(toolResult{Status: "success", Text: text})
}

func (r ToolRunner) executeSearchMemory(tc ai.ToolCall, chat *chatdomain.Chat) string {
	if !chat.Settings.Supermemory {
		return marshalToolResult(toolResult{Status: "error", Error: "supermemory is disabled in this chat"})
	}
	nodes := SearchMemoryNodes(chat, tc.Args["query"])
	if len(nodes) == 0 {
		return marshalToolResult(toolResult{Status: "success", Text: "No matches. Try a different keyword."})
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d node(s) matched (read_memory for details):\n", len(nodes)))
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("#%d — %s\n", n.ID, n.Hook))
	}
	return marshalToolResult(toolResult{Status: "success", Text: sb.String()})
}

func (r ToolRunner) executeReadMemory(tc ai.ToolCall, chat *chatdomain.Chat) string {
	if !chat.Settings.Supermemory {
		return marshalToolResult(toolResult{Status: "error", Error: "supermemory is disabled in this chat"})
	}
	node, errText := findMemoryNodeArg(chat, tc.Args["node_id"])
	if errText != "" {
		return marshalToolResult(toolResult{Status: "error", Error: errText})
	}
	return marshalToolResult(toolResult{Status: "success", Text: MemoryNodeForTool(chat, node)})
}

func (r ToolRunner) executeReadMemorySource(tc ai.ToolCall, chat *chatdomain.Chat) string {
	if !chat.Settings.Supermemory {
		return marshalToolResult(toolResult{Status: "error", Error: "supermemory is disabled in this chat"})
	}
	node, errText := findMemoryNodeArg(chat, tc.Args["node_id"])
	if errText != "" {
		return marshalToolResult(toolResult{Status: "error", Error: errText})
	}
	if node.To <= node.From {
		return marshalToolResult(toolResult{Status: "error", Error: fmt.Sprintf("node #%d has no raw transcript attached", node.ID)})
	}
	if r.Archive == nil {
		return marshalToolResult(toolResult{Status: "error", Error: "raw archive is not available"})
	}
	msgs, err := r.Archive.ReadRange(chat.ChatID, node.From, node.To)
	if err != nil {
		log.Printf("[ToolCall] read_memory_source error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: "failed to read the archive"})
	}
	return marshalToolResult(toolResult{Status: "success", Text: MemorySourceForTool(chat, msgs)})
}

// executeSaveMemory force-snapshots the archive's uncovered tail into a memory
// node, ignoring the accumulation threshold — the model's way to persist the
// conversation on demand.
func (r ToolRunner) executeSaveMemory(chat *chatdomain.Chat) string {
	if !chat.Settings.Supermemory {
		return marshalToolResult(toolResult{Status: "error", Error: "supermemory is disabled in this chat"})
	}
	if r.Compact == nil {
		return marshalToolResult(toolResult{Status: "error", Error: "memory snapshots are not available"})
	}
	session := chat.ActiveSession()
	ArchiveHistory(r.Archive, chat, session)
	before := len(chat.MemoryNodes)
	usage, err := r.Compact.Snapshot(chat, session.Model, true)
	if err != nil {
		log.Printf("[ToolCall] save_memory error: %v", err)
		return marshalToolResult(toolResult{Status: "error", Error: "failed to save memory, try again later"})
	}
	if usage != nil {
		chat.AccumulateCost(usage.Cost, usage.InputTokens, usage.OutputTokens)
	}
	if len(chat.MemoryNodes) == before {
		return marshalToolResult(toolResult{Status: "success", Text: "Nothing new to save — the conversation is already covered by memory nodes."})
	}
	n := chat.MemoryNodes[len(chat.MemoryNodes)-1]
	return marshalToolResult(toolResult{Status: "success", Text: fmt.Sprintf("Saved memory node #%d — %s", n.ID, n.Hook)})
}

// findMemoryNodeArg resolves a "#N" / "N" tool argument into a node.
func findMemoryNodeArg(chat *chatdomain.Chat, arg string) (*chatdomain.MemoryNode, string) {
	id, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(arg), "#")))
	if err != nil {
		return nil, "invalid node_id: " + arg
	}
	node := chat.FindMemoryNode(id)
	if node == nil {
		return nil, fmt.Sprintf("node #%d not found", id)
	}
	return node, ""
}

func (r ToolRunner) executeSaveNote(tc ai.ToolCall, chat *chatdomain.Chat) string {
	loc := chat.Location()
	remindAt, err := ParseRemindAt(tc.Args["remind_at"], loc, time.Now())
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	topic, err := AddAdvisorNote(chat, tc.Args["topic"], tc.Args["note"], remindAt)
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	text := fmt.Sprintf("Note saved to topic %q (%d entries)", topic.Name, len(topic.Entries))
	if remindAt != nil {
		text += ", reminder at " + remindAt.In(loc).Format("2006-01-02 15:04")
	}
	return marshalToolResult(toolResult{Status: "success", Text: text})
}

func (r ToolRunner) executeSetReminder(tc ai.ToolCall, chat *chatdomain.Chat) string {
	topic := chat.FindAdvisorTopicByName(tc.Args["topic"])
	if topic == nil {
		return marshalToolResult(toolResult{Status: "error", Error: "topic not found: " + tc.Args["topic"]})
	}
	entryID, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(tc.Args["entry_id"], "#")))
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: "invalid entry_id: " + tc.Args["entry_id"]})
	}
	loc := chat.Location()
	remindAt, err := ParseRemindAt(tc.Args["remind_at"], loc, time.Now())
	if err != nil {
		return marshalToolResult(toolResult{Status: "error", Error: err.Error()})
	}
	if !chat.SetAdvisorReminder(topic.ID, entryID, remindAt) {
		return marshalToolResult(toolResult{Status: "error", Error: fmt.Sprintf("entry #%d not found in topic %q", entryID, topic.Name)})
	}
	if remindAt == nil {
		return marshalToolResult(toolResult{Status: "success", Text: fmt.Sprintf("Reminder cleared on entry #%d", entryID)})
	}
	return marshalToolResult(toolResult{Status: "success", Text: fmt.Sprintf("Reminder on entry #%d set to %s", entryID, remindAt.In(loc).Format("2006-01-02 15:04"))})
}

func (r ToolRunner) executeReadNotes(tc ai.ToolCall, chat *chatdomain.Chat) string {
	topic := chat.FindAdvisorTopicByName(tc.Args["topic"])
	if topic == nil {
		return marshalToolResult(toolResult{Status: "error", Error: "topic not found: " + tc.Args["topic"]})
	}
	return marshalToolResult(toolResult{Status: "success", Text: AdvisorNotesForTool(topic, chat.Location())})
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
