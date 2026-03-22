package commands

// AllCommands returns every bot command, ready to use.
// To add a new command, append it here — no other changes needed.
func AllCommands(d *Deps) []Command {
	return []Command{
		&CommandHelp{Deps: d},
		&CommandStart{Deps: d},
		&CommandClear{Deps: d},
		&CommandHistory{Deps: d},
		&CommandRollback{Deps: d},
		&CommandTranslate{Deps: d},
		&CommandTechTranslate{Deps: d},
		&CommandEnhance{Deps: d},
		&CommandGrammar{Deps: d},
		&CommandSummarize{Deps: d},
		&CommandSummarizePrompt{Deps: d},
		&CommandAnalyze{Deps: d},
		&CommandModel{Deps: d},
		&CommandImagine{Deps: d},
		&CommandSystem{Deps: d},
		&CommandMarkdown{Deps: d},
		&CommandAdminReload{Deps: d},
		&CommandAdminAddUser{Deps: d},
		&CommandAdminRemoveUser{Deps: d},
		&CommandAutoReply{Deps: d},
		&CommandSessionList{Deps: d},
		&CommandSessionCurrent{Deps: d},
		&CommandSessionUse{Deps: d},
		&CommandSessionNew{Deps: d},
		&CommandSessionRemove{Deps: d},
		&CommandSessionUpdate{Deps: d},
	}
}

// RegisterAll populates the registry with every known command.
func RegisterAll(d *Deps) {
	for _, cmd := range AllCommands(d) {
		d.Registry.Add(cmd)
	}
}
