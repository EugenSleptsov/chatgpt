package commands

import (
	"GPTBot/application/service"
)

// Deps bundles every dependency the command set needs. Using a struct instead
// of a long positional argument list keeps wiring readable and makes adding a
// new dependency a non-breaking, field-only change.
type Deps struct {
	Registry        *Registry
	CmdService      *service.GPTCommandService
	ChatService     *service.ChatService
	Notifier        *service.Notifier
	Auth            *service.Auth
	History         *service.HistoryService
	Memory          *service.MemoryService
	ConfigService   *service.ConfigService
	ContextWindowFn func(string) int
}

// AllCommands returns every bot command, ready to use.
// Each command receives only the dependencies it actually needs.
func AllCommands(d Deps) []Command {
	return []Command{
		// --- zero-dep commands ---
		&CommandHelp{Registry: d.Registry, Auth: d.Auth},
		&CommandStart{},
		&CommandClear{History: d.History},
		&CommandHistory{History: d.History},
		&CommandRollback{History: d.History},
		&CommandModel{},
		&CommandSystem{},
		&CommandMarkdown{},
		&CommandAutoReply{},
		&CommandAutoRole{},
		&CommandMemory{Memory: d.Memory},
		&CommandSummarizePrompt{},
		&CommandUsage{},
		&CommandContext{ContextWindowFn: d.ContextWindowFn},
		&CommandSessionList{},
		&CommandSessionCurrent{},
		&CommandSessionUse{},
		&CommandSessionNew{},
		&CommandSessionRemove{},
		&CommandSessionUpdate{},

		// --- GPT text commands ---
		&CommandTranslate{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandTechTranslate{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandEnhance{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandGrammar{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandSummarize{Commands: d.CmdService, ChatService: d.ChatService, Notifier: d.Notifier},
		&CommandAnalyze{Commands: d.CmdService, ChatService: d.ChatService, Notifier: d.Notifier},

		// --- image ---
		&CommandImagine{Commands: d.CmdService, Notifier: d.Notifier, Auth: d.Auth},

		// --- admin ---
		&CommandAdminReload{ConfigService: d.ConfigService, Auth: d.Auth},
		&CommandAdminAddUser{ConfigService: d.ConfigService, Auth: d.Auth, Notifier: d.Notifier},
		&CommandAdminRemoveUser{ConfigService: d.ConfigService, Auth: d.Auth, Notifier: d.Notifier},
	}
}

// RegisterAll populates the registry with every known command.
func RegisterAll(d Deps) {
	for _, cmd := range AllCommands(d) {
		d.Registry.Add(cmd)
	}
}
