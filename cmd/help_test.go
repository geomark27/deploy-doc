package cmd

import "testing"

// TestEveryCommandHasHelp keeps commandHelp in sync with the commands map: a
// new command (or alias) without its own help fails here, instead of silently
// falling back to the general help and ignoring -h.
func TestEveryCommandHasHelp(t *testing.T) {
	for name := range commands {
		if _, ok := commandHelp[name]; !ok {
			t.Errorf("el comando %q no tiene ayuda en commandHelp", name)
		}
	}
	for name := range commandHelp {
		if _, ok := commands[name]; !ok {
			t.Errorf("commandHelp tiene %q, que no es un comando", name)
		}
	}
}

func TestIsHelpArg(t *testing.T) {
	for arg, want := range map[string]bool{"help": true, "-h": true, "--help": true, "-p": false, "": false} {
		if got := isHelpArg(arg); got != want {
			t.Errorf("isHelpArg(%q) = %v, want %v", arg, got, want)
		}
	}
}
