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
}

// Execute is the entry point for the CLI.
func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	cmdName := os.Args[1]
	if cmdName == "help" || cmdName == "--help" || cmdName == "-h" {
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

	return fn(os.Args[2:])
}

func printUsage() {
	fmt.Printf(clCyan+clBold+"gtt %s"+clReset+" — Generador de documentos de despliegue\n\n", build.Version)
	fmt.Print(clBold + "Uso:\n" + clReset)
	fmt.Print("  gtt <comando> [opciones]\n\n")
	fmt.Print(clBold + "Comandos:\n" + clReset)
	fmt.Print("  init              Configura tus credenciales de Atlassian\n")
	fmt.Print("  g, gen, generate  Genera un documento de despliegue en Confluence\n")
	fmt.Print("  qa                Genera consolidado de pruebas QA en Confluence\n")
	fmt.Print("  project           Gestiona proyectos (list, add, default, remove)\n")
	fmt.Print("  update            Actualiza gtt a la ultima version\n")
	fmt.Print("  version           Muestra la version actual\n\n")
	fmt.Print(clBold + "Flags de generate:\n" + clReset)
	fmt.Print("  -i, --issue            Clave del issue en Jira " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("  -b, --commit-backend   Hash(es) del commit de backend  (separar con coma)\n")
	fmt.Print("  -f, --commit-frontend  Hash(es) del commit de frontend (separar con coma)\n")
	fmt.Print("  -p, --project          Nombre del proyecto a usar (opcional)\n\n")
	fmt.Print(clBold + "Flags de qa:\n" + clReset)
	fmt.Print("  -s, --sprint           Número del sprint " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("  -m, --module           Módulo (ej: DAI, Aforo) " + clYellow + "(requerido)" + clReset + "\n\n")
	fmt.Print(clBold + "Ejemplos:\n" + clReset)
	fmt.Print("  gtt init\n")
	fmt.Print("  gtt g -i APP-1999 -b 27cefd86 -f 5bd0cea0\n")
	fmt.Print("  gtt qa -s 17 -m DAI\n")
	fmt.Print("  gtt project list\n")
	fmt.Print("  gtt update\n\n")
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
