package commands

import (
	"GPTBot/application/service"
)

// Deps bundles every dependency the command set needs. Using a struct instead
// of a long positional argument list keeps wiring readable and makes adding a
// new dependency a non-breaking, field-only change.
type Deps struct {
	Registry        *Registry
	CmdService      *service.GPTService
	ChatService     *service.ChatService
	Notifier        *service.Notifier
	Auth            *service.Auth
	ConfigService   *service.ConfigService
	ContextWindowFn func(string) int
	Progress        service.ProgressReporter // transient "Идет…" statuses (may be nil)
}

// AllCommands returns every bot command, ready to use.
// Each command receives only the dependencies it actually needs.
func AllCommands(d Deps) []Command {
	return []Command{
		// --- zero-dep commands ---
		&CommandHelp{Registry: d.Registry, Auth: d.Auth},
		&CommandMenu{},
		&CommandAdvisor{},
		&CommandSettings{Auth: d.Auth},
		&CommandStart{},
		&CommandClear{},
		&CommandHistory{},
		&CommandRollback{},
		&CommandSystem{},
		&CommandAutoRole{},
		&CommandSummarizePrompt{},
		&CommandUsage{},
		&CommandContext{ContextWindowFn: d.ContextWindowFn},
		&CommandSessionList{},
		&CommandSessionNew{},
		&CommandSessionRename{},
		&CommandSessionRemove{},

		// --- GPT text commands ---
		&CommandTranslate{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandTechTranslate{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandEnhance{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandGrammar{Commands: d.CmdService, Notifier: d.Notifier},
		&CommandSummarize{Commands: d.CmdService, ChatService: d.ChatService, Notifier: d.Notifier},
		&CommandAnalyze{Commands: d.CmdService, ChatService: d.ChatService, Notifier: d.Notifier},

		// --- image ---
		&CommandImagine{Commands: d.CmdService, Notifier: d.Notifier, Auth: d.Auth, Progress: d.Progress},

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
