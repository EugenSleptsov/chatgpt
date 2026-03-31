package commands

import (
	"GPTBot/application/service"
)

// AllCommands returns every bot command, ready to use.
// Each command receives only the dependencies it actually needs.
func AllCommands(
	registry *Registry,
	cmdService *service.GPTCommandService,
	chatService *service.ChatService,
	notifier *service.Notifier,
	auth *service.Auth,
	history *service.HistoryService,
	memory *service.MemoryService,
	configService *service.ConfigService,
	contextWindowFn func(string) int,
) []Command {
	return []Command{
		// --- zero-dep commands ---
		&CommandHelp{Registry: registry, Auth: auth},
		&CommandStart{},
		&CommandClear{History: history},
		&CommandHistory{History: history},
		&CommandRollback{History: history},
		&CommandModel{},
		&CommandSystem{},
		&CommandMarkdown{},
		&CommandAutoReply{},
		&CommandAutoRole{},
		&CommandMemory{Memory: memory},
		&CommandSummarizePrompt{},
		&CommandUsage{},
		&CommandContext{ContextWindowFn: contextWindowFn},
		&CommandSessionList{},
		&CommandSessionCurrent{},
		&CommandSessionUse{},
		&CommandSessionNew{},
		&CommandSessionRemove{},
		&CommandSessionUpdate{},

		// --- GPT text commands ---
		&CommandTranslate{Commands: cmdService, Notifier: notifier},
		&CommandTechTranslate{Commands: cmdService, Notifier: notifier},
		&CommandEnhance{Commands: cmdService, Notifier: notifier},
		&CommandGrammar{Commands: cmdService, Notifier: notifier},
		&CommandSummarize{Commands: cmdService, ChatService: chatService, Notifier: notifier},
		&CommandAnalyze{Commands: cmdService, ChatService: chatService, Notifier: notifier},

		// --- image ---
		&CommandImagine{Commands: cmdService, Notifier: notifier, Auth: auth},

		// --- admin ---
		&CommandAdminReload{ConfigService: configService, Auth: auth},
		&CommandAdminAddUser{ConfigService: configService, Auth: auth, Notifier: notifier},
		&CommandAdminRemoveUser{ConfigService: configService, Auth: auth, Notifier: notifier},
	}
}

// RegisterAll populates the registry with every known command.
func RegisterAll(
	registry *Registry,
	cmdService *service.GPTCommandService,
	chatService *service.ChatService,
	notifier *service.Notifier,
	auth *service.Auth,
	history *service.HistoryService,
	memory *service.MemoryService,
	configService *service.ConfigService,
	contextWindowFn func(string) int,
) {
	for _, cmd := range AllCommands(registry, cmdService, chatService, notifier, auth, history, memory, configService, contextWindowFn) {
		registry.Add(cmd)
	}
}
