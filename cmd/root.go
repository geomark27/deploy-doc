package cmd

import (
	"fmt"
	"os"

	"github.com/geomark27/deploy-doc/internal/build"
)

// commands registered here
var commands = map[string]func([]string) error{
	"init":     runInit,
	"generate": runGenerate,
	"update":   runUpdate,
	"project":  runProject,
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
		fmt.Printf("deploy-doc %s\n", build.Version)
		return nil
	}

	fn, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Comando desconocido: %s\n\n", cmdName)
		printUsage()
		return fmt.Errorf("comando inválido")
	}

	return fn(os.Args[2:])
}

func printUsage() {
	fmt.Printf(`deploy-doc %s - Generador de documentos de despliegue

Uso:
  deploy-doc <comando> [opciones]

Comandos:
  init      Configura tus credenciales de Atlassian
  generate  Genera un documento de despliegue en Confluence
  project   Gestiona proyectos (list, add, default, remove)
  update    Actualiza deploy-doc a la ultima version
  version   Muestra la version actual

Flags de generate:
  --issue            Clave del issue en Jira (requerido)
  --commit-backend   Hash del commit de backend
  --commit-frontend  Hash del commit de frontend
  --project          Nombre del proyecto a usar (opcional)

Ejemplos:
  deploy-doc init
  deploy-doc generate --issue APP-1999 --commit-backend 27cefd86 --commit-frontend 5bd0cea0
  deploy-doc generate --project ecuapass --issue ECU-123 --commit-backend abc1234
  deploy-doc project list
  deploy-doc project add
  deploy-doc project default echo
  deploy-doc update
`, build.Version)
}
