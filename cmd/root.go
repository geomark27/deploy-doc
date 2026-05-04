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

	// ── Uso ──────────────────────────────────────────────────────────────────
	fmt.Print(clBold + "Uso:\n" + clReset)
	fmt.Print("  gtt <comando> [flags]\n\n")

	// ── Comandos ─────────────────────────────────────────────────────────────
	fmt.Print(clBold + "Comandos:\n" + clReset)
	fmt.Print("  init              Configura credenciales y preferencias (wizard interactivo)\n")
	fmt.Print("  g, gen, generate  Genera un documento de despliegue en Confluence\n")
	fmt.Print("  qa                Genera consolidado de pruebas QA en Confluence\n")
	fmt.Print("  project           Gestiona proyectos: list, add, default, remove\n")
	fmt.Print("  update            Actualiza gtt a la última versión\n")
	fmt.Print("  version           Muestra la versión instalada\n\n")

	// ── gtt init ─────────────────────────────────────────────────────────────
	fmt.Print(clBold + "gtt init — qué configura:\n" + clReset)
	fmt.Print("  Atlassian email         Tu email de la cuenta Atlassian\n")
	fmt.Print("  Atlassian API token     Token generado en id.atlassian.com\n")
	fmt.Print("  Atlassian base URL      URL de tu instancia (ej: https://empresa.atlassian.net)\n")
	fmt.Print("  Confluence space key    Space por defecto (ej: PA). Opcional — puedes\n")
	fmt.Print("                          cambiarlo por comando con --space\n")
	fmt.Print("  Al configurar un proyecto también pide:\n")
	fmt.Print("    Rutas locales backend/frontend\n")
	fmt.Print("    Nombres de repositorios\n")
	fmt.Print("    VCS host  (ej: https://bitbucket.org)\n")
	fmt.Print("    VCS org   (ej: devtyt)\n\n")
	fmt.Print("  Todo se guarda en " + clCyan + "~/.config/deploy-doc/config.yaml" + clReset + "\n")
	fmt.Print("  Las variables de entorno tienen prioridad sobre el archivo:\n")
	fmt.Print("    ATLASSIAN_EMAIL, ATLASSIAN_TOKEN, ATLASSIAN_BASE_URL, CONFLUENCE_SPACE_KEY\n\n")

	// ── generate ─────────────────────────────────────────────────────────────
	fmt.Print(clBold + "Flags de generate (gtt g):\n" + clReset)
	fmt.Print("  -i, --issue            Clave del issue en Jira              " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("  -b, --commit-backend   Hash(es) de commits backend          (separar con coma)\n")
	fmt.Print("  -f, --commit-frontend  Hash(es) de commits frontend         (separar con coma)\n")
	fmt.Print("  -p, --project          Proyecto a usar (del config.yaml)\n")
	fmt.Print("  -s, --space            " + clCyan + "Override" + clReset + " del Confluence space key para esta ejecución\n")
	fmt.Print("      --vcs-host         " + clCyan + "Override" + clReset + " del host VCS para esta ejecución\n")
	fmt.Print("      --vcs-org          " + clCyan + "Override" + clReset + " de la org/workspace VCS para esta ejecución\n\n")
	fmt.Print("  Prioridad de " + clBold + "--space" + clReset + ":\n")
	fmt.Print("    1. Flag --space en el comando                (máxima prioridad)\n")
	fmt.Print("    2. confluence_space_key del proyecto (-p)\n")
	fmt.Print("    3. confluence_space_key global en config.yaml\n\n")
	fmt.Print("  Prioridad de " + clBold + "--vcs-host / --vcs-org" + clReset + ":\n")
	fmt.Print("    1. Flags --vcs-host / --vcs-org en el comando\n")
	fmt.Print("    2. vcs_host / vcs_org del proyecto en config.yaml\n\n")

	// ── qa ───────────────────────────────────────────────────────────────────
	fmt.Print(clBold + "Flags de qa:\n" + clReset)
	fmt.Print("  -s, --sprint           Número del sprint                    " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("  -m, --module           Módulo (ej: DAI, Aforo)              " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("      --space            " + clCyan + "Override" + clReset + " del Confluence space key para esta ejecución\n\n")
	fmt.Print("  Prioridad de " + clBold + "--space" + clReset + " (misma lógica que generate):\n")
	fmt.Print("    1. Flag --space en el comando\n")
	fmt.Print("    2. confluence_space_key global en config.yaml\n\n")

	// ── Ejemplos ─────────────────────────────────────────────────────────────
	fmt.Print(clBold + "Ejemplos:\n" + clReset)
	fmt.Print("\n  " + clBold + "# Uso estándar (space key viene del config.yaml):" + clReset + "\n")
	fmt.Print("  gtt g -i APP-1999 -b 27cefd86 -f 5bd0cea0\n")
	fmt.Print("  gtt qa -s 17 -m DAI\n")
	fmt.Print("\n  " + clBold + "# Override puntual de space key (sin editar config.yaml):" + clReset + "\n")
	fmt.Print("  gtt g -i ADN-567 -b abc1234 --space ADN\n")
	fmt.Print("  gtt qa -s 42 -m Aforo --space ADN\n")
	fmt.Print("\n  " + clBold + "# Override puntual de VCS (ej: repo en otro workspace):" + clReset + "\n")
	fmt.Print("  gtt g -i APP-1999 -b abc1234 --vcs-org mi-fork --vcs-host https://github.com\n")
	fmt.Print("\n  " + clBold + "# Gestión de proyectos:" + clReset + "\n")
	fmt.Print("  gtt project list\n")
	fmt.Print("  gtt g -i APP-1999 -b abc1234 -p echo\n")
	fmt.Print("\n  " + clBold + "# Otros:" + clReset + "\n")
	fmt.Print("  gtt init\n")
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
