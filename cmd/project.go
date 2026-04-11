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
	"ls":      runProjectList,
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
		fmt.Fprintf(os.Stderr, clr(clRed, "Subcomando desconocido: ")+"project %s\n\n", sub)
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
		fmt.Println("Usa: gtt project add")
		return nil
	}

	fmt.Printf(clBold+"%-15s %-45s %-45s"+clReset+"\n", "PROYECTO", "BACKEND PATH", "FRONTEND PATH")
	fmt.Println(strings.Repeat("─", 108))
	for name, proj := range cfg.Projects {
		isDefault := name == cfg.DefaultProject
		nameStr := name
		if isDefault {
			nameStr = clr(clGreen+clBold, "* "+name)
		} else {
			nameStr = "  " + name
		}
		backendPath := proj.BackendPath
		if backendPath == "" {
			backendPath = clr(clYellow, "(no configurado)")
		}
		frontendPath := proj.FrontendPath
		if frontendPath == "" {
			frontendPath = clr(clYellow, "(no configurado)")
		}
		fmt.Printf("%-15s %-45s %-45s\n", nameStr, backendPath, frontendPath)
	}
	fmt.Println()
	if cfg.DefaultProject != "" {
		fmt.Printf(clGreen+"*"+clReset+" Proyecto por defecto: "+clBold+"%s"+clReset+"\n", cfg.DefaultProject)
	}
	return nil
}

func runProjectAdd(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println(clCyan + clBold + "── Agregar proyecto ──" + clReset)
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

	fmt.Printf("\n"+clGreen+"✓"+clReset+" Proyecto '%s' guardado\n", name)
	return nil
}

func runProjectDefault(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: gtt project default <nombre>")
	}
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("proyecto '%s' no encontrado. Usa: gtt project list", name)
	}

	cfg.DefaultProject = name
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	fmt.Printf(clGreen+"✓"+clReset+" Proyecto por defecto cambiado a "+clBold+"'%s'"+clReset+"\n", name)
	return nil
}

func runProjectRemove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: gtt project remove <nombre>")
	}
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Projects[name]; !ok {
		return fmt.Errorf("proyecto '%s' no encontrado. Usa: gtt project list", name)
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
		warnLine(fmt.Sprintf("'%s' era el proyecto por defecto. Usa: gtt project default <nombre>", name))
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	fmt.Printf(clGreen+"✓"+clReset+" Proyecto '%s' eliminado\n", name)
	return nil
}

func printProjectUsage() {
	fmt.Printf(clCyan + clBold + "gtt project" + clReset + " — Gestiona proyectos configurados\n\n")
	fmt.Print(clBold + "Uso:\n" + clReset)
	fmt.Print("  gtt project <subcomando> [opciones]\n\n")
	fmt.Print(clBold + "Subcomandos:\n" + clReset)
	fmt.Print("  list, ls          Lista los proyectos configurados\n")
	fmt.Print("  add               Agrega un nuevo proyecto (asistente interactivo)\n")
	fmt.Print("  default <nombre>  Cambia el proyecto por defecto\n")
	fmt.Print("  remove <nombre>   Elimina un proyecto\n\n")
	fmt.Print(clBold + "Ejemplos:\n" + clReset)
	fmt.Print("  gtt project list\n")
	fmt.Print("  gtt project add\n")
	fmt.Print("  gtt project default echo\n")
	fmt.Print("  gtt project remove ecuapass\n")
}
