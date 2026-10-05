package cmd

import (
	"fmt"
	"runtime"

	"github.com/geomark27/deploy-doc/internal/build"
)

// commandHelp maps every command (and its aliases) to its detailed help, shown
// by `gtt help <comando>` and by `gtt <comando> -h`. TestEveryCommandHasHelp
// keeps it in sync with the commands map.
var commandHelp = map[string]func(){
	"init":     printInitUsage,
	"generate": printGenerateUsage,
	"gen":      printGenerateUsage,
	"g":        printGenerateUsage,
	"qa":       printQAUsage,
	"fetch":    printFetchUsage,
	"f":        printFetchUsage,
	"project":  printProjectUsage,
	"backlog":  printBacklogUsage,
	"skill":    printSkillUsage,
	"update":   printUpdateUsage,
}

// isHelpArg reports whether arg asks for help.
func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

// wantsHelp reports whether any argument asks for help with -h or --help.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

// helpTitle prints the first line of a command's help.
func helpTitle(name, summary string) {
	fmt.Print(clCyan + clBold + name + clReset + " — " + summary + "\n\n")
}

// helpSection prints a section heading of a help screen.
func helpSection(title string) {
	fmt.Print(clBold + title + clReset + "\n")
}

// configPathHint shows where config.yaml lives, written the way the user's
// system writes paths.
func configPathHint() string {
	if runtime.GOOS == "windows" {
		return `%USERPROFILE%\.config\gtt\config.yaml`
	}
	return "~/.config/gtt/config.yaml"
}

// printUsage is the general help: one line per command, grouped by purpose.
// The detail of each command lives in its own help (gtt help <comando>).
func printUsage() {
	fmt.Printf(clCyan+clBold+"gtt %s"+clReset+" — Documentación de despliegues en Atlassian y backlog técnico\n\n", build.Version)

	helpSection("Uso:")
	fmt.Print("  gtt <comando> [flags]\n")
	fmt.Print("  gtt help <comando>      Ayuda detallada de un comando (también: gtt <comando> -h)\n\n")

	helpSection("Configuración:")
	fmt.Print("  init              Configura las credenciales de Atlassian y el primer proyecto\n")
	fmt.Print("  project           Gestiona los proyectos: list, add, default, remove\n\n")

	helpSection("Documentación en Atlassian:")
	fmt.Print("  g, gen, generate  Crea o actualiza el documento de despliegue de una tarea\n")
	fmt.Print("  qa                Publica el consolidado de pruebas QA (Sprint o Kanban)\n")
	fmt.Print("  f, fetch          Exporta a .txt la página de Confluence de una tarea\n\n")

	helpSection("Backlog técnico:")
	fmt.Print("  backlog scan      Detecta deuda técnica y la propone como tareas (sin credenciales)\n")
	fmt.Print("  skill             Skill backlog-tareas para Claude Code: status, install, diff, reset\n\n")

	helpSection("Mantenimiento:")
	fmt.Print("  update            Actualiza gtt a la última versión\n")
	fmt.Print("  version           Muestra la versión instalada\n\n")

	helpSection("Ejemplos rápidos:")
	fmt.Print("  gtt g -i APP-1999 -b 27cefd86          # documento de despliegue de APP-1999\n")
	fmt.Print("  gtt qa                                 # consolidado QA de los últimos 10 días hábiles\n")
	fmt.Print("  gtt f -i APP-1981                      # exportar la página de APP-1981 a .txt\n")
	fmt.Print("  gtt backlog scan                       # tareas de deuda técnica de tus repos\n")
	fmt.Print("  gtt help g                             # todo sobre generate\n\n")

	fmt.Print("Configuración en " + clCyan + configPathHint() + clReset + ".\n")
	fmt.Print("Las variables ATLASSIAN_EMAIL, ATLASSIAN_TOKEN, ATLASSIAN_BASE_URL y\n")
	fmt.Print("CONFLUENCE_SPACE_KEY tienen prioridad sobre el archivo.\n")
}

func printInitUsage() {
	helpTitle("gtt init", "Configura las credenciales de Atlassian y el primer proyecto")

	helpSection("Uso:")
	fmt.Print("  gtt init\n\n")
	fmt.Print("  Asistente interactivo, sin flags. Valida el token contra Atlassian antes de\n")
	fmt.Print("  guardar. Si ya existe una configuración, pregunta antes de sobrescribirla\n")
	fmt.Print("  (por defecto No); los proyectos configurados se conservan siempre.\n\n")

	helpSection("Qué configura:")
	fmt.Print("  Atlassian email         Tu email de la cuenta Atlassian\n")
	fmt.Print("  Atlassian API token     Token generado en id.atlassian.com\n")
	fmt.Print("  Atlassian base URL      URL de tu instancia (ej: https://empresa.atlassian.net)\n")
	fmt.Print("  Confluence space key    Space por defecto (ej: PA). Opcional: se puede\n")
	fmt.Print("                          cambiar por comando con --space\n")
	fmt.Print("  Al configurar un proyecto también pide:\n")
	fmt.Print("    Rutas locales backend/frontend\n")
	fmt.Print("    Nombres de repositorios\n")
	fmt.Print("    VCS host  (ej: https://bitbucket.org)\n")
	fmt.Print("    VCS org   (ej: mi-organizacion)\n\n")

	helpSection("Dónde se guarda:")
	fmt.Print("  " + clCyan + configPathHint() + clReset + "\n")
	fmt.Print("  Las variables de entorno tienen prioridad sobre el archivo:\n")
	fmt.Print("    ATLASSIAN_EMAIL, ATLASSIAN_TOKEN, ATLASSIAN_BASE_URL, CONFLUENCE_SPACE_KEY\n\n")

	helpSection("Ver también:")
	fmt.Print("  gtt help project        Agregar o cambiar proyectos sin repetir init\n")
}

func printGenerateUsage() {
	helpTitle("gtt g, gen, generate", "Crea o actualiza el documento de despliegue de una tarea")

	helpSection("Uso:")
	fmt.Print("  gtt g -i <ISSUE> [-b <HASH>] [-f <HASH>] [flags]\n\n")

	helpSection("Flags:")
	fmt.Print("  -i, --issue            Clave del issue en Jira              " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("  -b, --commit-backend   Hash(es) de commits backend          (separar con coma)\n")
	fmt.Print("  -f, --commit-frontend  Hash(es) de commits frontend         (separar con coma)\n")
	fmt.Print("  -p, --project          Proyecto a usar (del config.yaml)\n")
	fmt.Print("  -s, --space            " + clCyan + "Override" + clReset + " del Confluence space key para esta ejecución\n")
	fmt.Print("      --vcs-host         " + clCyan + "Override" + clReset + " del host VCS para esta ejecución\n")
	fmt.Print("      --vcs-org          " + clCyan + "Override" + clReset + " de la org/workspace VCS para esta ejecución\n")
	fmt.Print("      --dry-run          Imprime el documento (ADF) sin publicar nada en Confluence\n")
	fmt.Print("  -h, --help             Muestra esta ayuda\n\n")
	fmt.Print("  Al menos uno de -b o -f es requerido. Se aceptan -i APP-1, -i=APP-1 y --issue APP-1.\n\n")

	helpSection("Prioridad de --space:")
	fmt.Print("  1. Flag --space en el comando                (máxima prioridad)\n")
	fmt.Print("  2. confluence_space_key del proyecto (-p)\n")
	fmt.Print("  3. confluence_space_key global en config.yaml\n\n")

	helpSection("Prioridad de --vcs-host / --vcs-org:")
	fmt.Print("  1. Flags --vcs-host / --vcs-org en el comando\n")
	fmt.Print("  2. vcs_host / vcs_org del proyecto en config.yaml\n\n")

	helpSection("Sección \"A considerar\":")
	fmt.Print("  Al " + clBold + "crear" + clReset + "      — pasos de deploy_checklist (proyecto > global > default genérico)\n")
	fmt.Print("  Al " + clBold + "actualizar" + clReset + " — se preserva tal como quedó editada en Confluence\n\n")

	helpSection("Ejemplos:")
	fmt.Print("  gtt g -i APP-1999 -b 27cefd86 -f 5bd0cea0\n")
	fmt.Print("  gtt g -i APP-1999 -b abc123,def456            # varios commits\n")
	fmt.Print("  gtt g -i APP-1999 -b abc1234 -p echo          # otro proyecto\n")
	fmt.Print("  gtt g -i ADN-567 -b abc1234 --space ADN       # otro space, sin editar config.yaml\n")
	fmt.Print("  gtt g -i APP-1999 -b abc1234 --vcs-org mi-fork --vcs-host https://github.com\n")
	fmt.Print("  gtt g -i APP-1999 -b 27cefd86 --dry-run       # vista previa sin publicar\n")
}

func printQAUsage() {
	helpTitle("gtt qa", "Publica el consolidado de pruebas QA en Confluence")

	helpSection("Uso:")
	fmt.Print("  gtt qa [-s <SPRINT> -m <MÓDULO>] [flags]\n\n")

	helpSection("Flags:")
	fmt.Print("  -s, --sprint   Número del sprint                    (opcional)\n")
	fmt.Print("  -m, --module   Módulo (ej: DAI, Aforo)              (solo Sprint)\n")
	fmt.Print("      --space    " + clCyan + "Override" + clReset + " del Confluence space key para esta ejecución\n")
	fmt.Print("      --dry-run  Imprime el reporte (ADF) sin publicar nada en Confluence\n")
	fmt.Print("  -h, --help     Muestra esta ayuda\n\n")

	helpSection("Modos de operación:")
	fmt.Print("  Sprint  — con -s y -m: busca las tareas del módulo en el sprint indicado\n")
	fmt.Print("  Kanban  — sin -s ni -m: busca " + clBold + "todas" + clReset + " las tareas de los últimos 10 días hábiles\n\n")

	helpSection("Prioridad de --space:")
	fmt.Print("  1. Flag --space en el comando\n")
	fmt.Print("  2. confluence_space_key global en config.yaml\n\n")

	helpSection("Encabezado del reporte:")
	fmt.Print("  Se edita en " + clCyan + configPathHint() + clReset + ":\n")
	fmt.Print("    qa_report:\n")
	fmt.Print("      lider_tecnico: \"...\"\n")
	fmt.Print("      pmo: \"...\"\n")
	fmt.Print("      qa: \"...\"\n")
	fmt.Print("  Las claves ausentes se publican como " + clBold + "—" + clReset + "\n\n")

	helpSection("Ejemplos:")
	fmt.Print("  gtt qa                          # modo Kanban: todas las tareas, 10 días hábiles\n")
	fmt.Print("  gtt qa -s 17 -m DAI             # modo Sprint\n")
	fmt.Print("  gtt qa -s 42 -m Aforo --space ADN\n")
	fmt.Print("  gtt qa -s 17 -m DAI --dry-run   # vista previa sin publicar\n")
}

func printFetchUsage() {
	helpTitle("gtt f, fetch", "Exporta a .txt la página de Confluence de una tarea")

	helpSection("Uso:")
	fmt.Print("  gtt f -i <ISSUE> [flags]\n\n")
	fmt.Print("  Busca la página por la clave del issue, la convierte a texto plano (títulos,\n")
	fmt.Print("  listas, tablas, tareas y paneles) y abre un diálogo para elegir dónde guardarla.\n\n")

	helpSection("Flags:")
	fmt.Print("  -i, --issue    Clave del issue en Jira              " + clYellow + "(requerido)" + clReset + "\n")
	fmt.Print("  -o, --output   Nombre del archivo de salida         (por defecto: APP-1981_Titulo.txt)\n")
	fmt.Print("  -s, --space    " + clCyan + "Override" + clReset + " del Confluence space key para esta búsqueda\n")
	fmt.Print("  -h, --help     Muestra esta ayuda\n\n")

	helpSection("Ejemplos:")
	fmt.Print("  gtt f -i APP-1981\n")
	fmt.Print("  gtt f -i APP-1981 -o mi_tarea.txt\n")
	fmt.Print("  gtt f -i APP-1981 --space PA\n")
}

func printUpdateUsage() {
	helpTitle("gtt update", "Actualiza gtt a la última versión")

	helpSection("Uso:")
	fmt.Print("  gtt update\n\n")
	fmt.Print("  Descarga el último release de GitHub, verifica su SHA-256 contra checksums.txt\n")
	fmt.Print("  y reemplaza el binario instalado. Sin flags.\n\n")
	fmt.Print("  gtt avisa al ejecutar cualquier comando cuando hay una versión nueva (como mucho\n")
	fmt.Print("  una consulta cada 24 h). Para desactivar el aviso: GTT_NO_UPDATE_CHECK=1\n")
}
