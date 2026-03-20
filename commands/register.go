package commands

// Constructors returns all available command constructors.
// Each constructor receives *Deps and returns a ready-to-use Command.
// To add a new command, append it here — no changes in main.go needed.
func Constructors() []func(d *Deps) Command {
	return []func(d *Deps) Command{
		func(d *Deps) Command { return &CommandHelp{Deps: d} },
		func(d *Deps) Command { return &CommandStart{Deps: d} },
		func(d *Deps) Command { return &CommandClear{Deps: d} },
		func(d *Deps) Command { return &CommandHistory{Deps: d} },
		func(d *Deps) Command { return &CommandRollback{Deps: d} },
		func(d *Deps) Command { return &CommandTranslate{Deps: d} },
		func(d *Deps) Command { return &CommandTechTranslate{Deps: d} },
		func(d *Deps) Command { return &CommandEnhance{Deps: d} },
		func(d *Deps) Command { return &CommandGrammar{Deps: d} },
		func(d *Deps) Command { return &CommandSummarize{Deps: d} },
		func(d *Deps) Command { return &CommandSummarizePrompt{Deps: d} },
		func(d *Deps) Command { return &CommandAnalyze{Deps: d} },
		func(d *Deps) Command { return &CommandModel{Deps: d} },
		func(d *Deps) Command { return &CommandImagine{Deps: d} },
		func(d *Deps) Command { return &CommandSystem{Deps: d} },
		func(d *Deps) Command { return &CommandMarkdown{Deps: d} },
		func(d *Deps) Command { return &CommandAdminReload{Deps: d} },
		func(d *Deps) Command { return &CommandAdminAddUser{Deps: d} },
		func(d *Deps) Command { return &CommandAdminRemoveUser{Deps: d} },
		func(d *Deps) Command { return &CommandSessionList{Deps: d} },
		func(d *Deps) Command { return &CommandSessionCurrent{Deps: d} },
		func(d *Deps) Command { return &CommandSessionUse{Deps: d} },
		func(d *Deps) Command { return &CommandSessionNew{Deps: d} },
		func(d *Deps) Command { return &CommandSessionRemove{Deps: d} },
		func(d *Deps) Command { return &CommandSessionUpdate{Deps: d} },
	}
}

// RegisterAll registers every known command with the given Deps.
func RegisterAll(d *Deps) {
	for _, ctor := range Constructors() {
		cmd := ctor(d)
		d.Registry.Register(cmd.Name(), func(c Command) func() Command {
			return func() Command { return c }
		}(cmd))
	}
}
