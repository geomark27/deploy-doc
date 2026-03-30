package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/geomark27/deploy-doc/internal/config"
)

var projectSubcommands = map[string]func([]string) error{
	"list":    runProjectList,
	"add":     runProjectAdd,
	"default": runProjectDefault,
	"remove":  runProjectRemove,
}

func runProject(args []string) error {
	if len(args) == 0 {
		printProjectUsage()
		return nil
	}
	sub := args[0]
	fn, ok := projectSubcommands[sub]
	if !ok {
		fmt.Fprintf(os.Stderr, "Subcomando desconocido: project %s\n\n", sub)
		printProjectUsage()
		return fmt.Errorf("subcomando inválido")
	}
	return fn(args[1:])
}

func runProjectList(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Projects) == 0 {
		fmt.Println("No hay proyectos configurados.")
		fmt.Println("Usa: deploy-doc project add")
		return nil
	}

	fmt.Printf("%-15s %-45s %-45s\n", "PROYECTO", "BACKEND PATH", "FRONTEND PATH")
	fmt.Println(strings.Repeat("-", 108))
	for name, proj := range cfg.Projects {
		marker := "  "
		if name == cfg.DefaultProject {
			marker = "* "
		}
		backendPath := proj.BackendPath
		if backendPath == "" {
			backendPath = "(no configurado)"
		}
		frontendPath := proj.FrontendPath
		if frontendPath == "" {
			frontendPath = "(no configurado)"
		}
		fmt.Printf("%s%-13s %-45s %-45s\n", marker, name, backendPath, frontendPath)
	}
	fmt.Println()
	if cfg.DefaultProject != "" {
		fmt.Printf("* Proyecto por defecto: %s\n", cfg.DefaultProject)
	}
	return nil
}

func runProjectAdd(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Agregar proyecto ---")
	fmt.Println()

	name, err := prompt(reader, "Nombre del proyecto (ej: echo)")
	if err != nil {
		return err
	}

	if _, exists := cfg.Projects[name]; exists {
		fmt.Printf("El proyecto '%s' ya existe.\n", name)
		fmt.Print("¿Deseas sobreescribirlo? [s/N]: ")
		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans != "s" && ans != "si" && ans != "sí" {
			fmt.Println("Cancelado.")
			return nil
		}
	}

	proj := &config.ProjectConfig{}

	proj.BackendPath, err = promptOptional(reader, "Ruta del repositorio backend")
	if err != nil {
		return err
	}
	if proj.BackendPath != "" {
		proj.BackendRepo, err = promptWithDefault(reader, "Nombre del repositorio backend", name+"-api")
		if err != nil {
			return err
		}
	}

	proj.FrontendPath, err = promptOptional(reader, "Ruta del repositorio frontend")
	if err != nil {
		return err
	}
	if proj.FrontendPath != "" {
		proj.FrontendRepo, err = promptWithDefault(reader, "Nombre del repositorio frontend", name)
		if err != nil {
			return err
		}
	}

	if proj.BackendPath == "" && proj.FrontendPath == "" {
		fmt.Println("No se configuraron rutas. Proyecto omitido.")
		return nil
	}

	if cfg.Projects == nil {
		cfg.Projects = make(map[string]*config.ProjectConfig)
	}
	cfg.Projects[name] = proj

	fmt.Print("¿Establecer como proyecto por defecto? [S/n]: ")
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans == "" || ans == "s" || ans == "si" || ans == "sí" {
		cfg.DefaultProject = name
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando proyecto: %w", err)
	}

	fmt.Printf("\n✓ Proyecto '%s' guardado\n", name)
	return nil
}

func runProjectDefault(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: deploy-doc project default <nombre>")
	}
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("proyecto '%s' no encontrado. Usa: deploy-doc project list", name)
	}

	cfg.DefaultProject = name
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	fmt.Printf("✓ Proyecto por defecto cambiado a '%s'\n", name)
	return nil
}

func runProjectRemove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: deploy-doc project remove <nombre>")
	}
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("proyecto '%s' no encontrado. Usa: deploy-doc project list", name)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("¿Eliminar el proyecto '%s'? [s/N]: ", name)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans != "s" && ans != "si" && ans != "sí" {
		fmt.Println("Cancelado.")
		return nil
	}

	delete(cfg.Projects, name)

	if cfg.DefaultProject == name {
		cfg.DefaultProject = ""
		fmt.Printf("Advertencia: '%s' era el proyecto por defecto. Usa: deploy-doc project default <nombre>\n", name)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	fmt.Printf("✓ Proyecto '%s' eliminado\n", name)
	return nil
}

func printProjectUsage() {
	fmt.Println(`deploy-doc project - Gestiona proyectos configurados

Uso:
  deploy-doc project <subcomando> [opciones]

Subcomandos:
  list              Lista los proyectos configurados
  add               Agrega un nuevo proyecto (asistente interactivo)
  default <nombre>  Cambia el proyecto por defecto
  remove <nombre>   Elimina un proyecto

Ejemplos:
  deploy-doc project list
  deploy-doc project add
  deploy-doc project default echo
  deploy-doc project remove ecuapass`)
}
