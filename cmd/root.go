package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/geomark27/deploy-doc/internal/build"
)

// ANSI color helpers — used across all cmd files.
const (
	clReset  = "\033[0m"
	clBold   = "\033[1m"
	clRed    = "\033[31m"
	clGreen  = "\033[32m"
	clYellow = "\033[33m"
	clCyan   = "\033[36m"
)

func clr(color, text string) string { return color + text + clReset }

// stepLabel prints a cyan "[n/total] msg" line.
func stepLabel(n, total int, msg string) {
	fmt.Printf("%s[%d/%d]%s %s\n", clCyan+clBold, n, total, clReset, msg)
}

// okLine prints an indented green ✓ line.
func okLine(msg string) { fmt.Printf("      %s✓%s %s\n", clGreen, clReset, msg) }

// warnLine prints an indented yellow ⚠ line.
func warnLine(msg string) { fmt.Printf("      %s⚠%s %s\n", clYellow, clReset, msg) }

// errLine prints an indented red ✗ line.
func errLine(msg string) { fmt.Printf("      %s✗%s %s\n", clRed, clReset, msg) }

// commands registered here — g and gen are short aliases for generate.
var commands = map[string]func([]string) error{
	"init":     runInit,
	"generate": runGenerate,
	"gen":      runGenerate,
	"g":        runGenerate,
	"update":   runUpdate,
	"project":  runProject,
	"qa":       runQA,
	"fetch":    runFetch,
	"f":        runFetch,
	"backlog":  runBacklog,
}

// Execute is the entry point for the CLI.
func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	cmdName := os.Args[1]
	if isHelpArg(cmdName) {
		// gtt help <comando> shows that command's own help when it has one.
		if len(os.Args) > 2 {
			if fn, ok := commandHelp[os.Args[2]]; ok {
				fn()
				return nil
			}
		}
		printUsage()
		return nil
	}
	if cmdName == "version" || cmdName == "--version" || cmdName == "-v" {
		fmt.Printf("gtt %s\n", build.Version)
		return nil
	}

	fn, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, clr(clRed, "Comando desconocido: ")+"%s\n\n", cmdName)
		printUsage()
		return fmt.Errorf("comando inválido")
	}

	// -h / --help is answered here, before the command runs: otherwise a
	// command that ignores flags (init, update) would do its job when the
	// user only asked how to use it.
	if help, ok := commandHelp[cmdName]; ok && wantsHelp(os.Args[2:]) {
		help()
		return nil
	}

	return fn(os.Args[2:])
}

// parseFlagsWithShorts normalizes short flags to their long form then parses all flags.
func parseFlagsWithShorts(args []string, shorts map[string]string) map[string]string {
	normalized := make([]string, len(args))
	for i, a := range args {
		if !strings.HasPrefix(a, "--") && strings.HasPrefix(a, "-") {
			if idx := strings.IndexByte(a, '='); idx != -1 {
				short := a[:idx]
				if long, ok := shorts[short]; ok {
					normalized[i] = long + a[idx:]
					continue
				}
			} else if long, ok := shorts[a]; ok {
				normalized[i] = long
				continue
			}
		}
		normalized[i] = a
	}

	flags := make(map[string]string)
	i := 0
	for i < len(normalized) {
		arg := normalized[i]
		if !strings.HasPrefix(arg, "--") {
			i++
			continue
		}
		if idx := strings.IndexByte(arg, '='); idx != -1 {
			flags[arg[:idx]] = arg[idx+1:]
			i++
		} else if i+1 < len(normalized) && !strings.HasPrefix(normalized[i+1], "--") && !strings.HasPrefix(normalized[i+1], "-") {
			flags[arg] = normalized[i+1]
			i += 2
		} else {
			flags[arg] = ""
			i++
		}
	}
	return flags
}
