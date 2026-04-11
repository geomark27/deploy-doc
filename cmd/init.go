package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/geomark27/deploy-doc/internal/config"
)

func runInit(args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Warn if config already exists before overwriting
	if path, err := config.ConfigPath(); err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			fmt.Printf("\nYa existe una configuración en %s\n", path)
			fmt.Print("¿Sobreescribir? [s/N]: ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "s" && answer != "si" && answer != "sí" {
				fmt.Println("Cancelado. Configuración sin cambios.")
				return nil
			}
		}
	}

	fmt.Println(clCyan + clBold + "── Configuración de gtt ──" + clReset)
	fmt.Println("Tus credenciales se guardarán en ~/.config/deploy-doc/config.yaml")
	fmt.Println()

	email, err := prompt(reader, "Atlassian email")
	if err != nil {
		return err
	}

	token, err := prompt(reader, "Atlassian API token (https://id.atlassian.com/manage-profile/security/api-tokens)")
	if err != nil {
		return err
	}

	// Load existing config to preserve projects if any
	existing, _ := config.Load()

	cfg := &config.Config{
		AtlassianEmail: email,
		AtlassianToken: token,
		BaseURL:        "https://torresytorres.atlassian.net",
	}
	if existing != nil {
		cfg.DefaultProject = existing.DefaultProject
		cfg.Projects = existing.Projects
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	path, _ := config.ConfigPath()
	fmt.Printf("\n"+clGreen+"✓"+clReset+" Configuración guardada en %s\n", path)

	// --- Optional: configure first project ---
	fmt.Println()
	fmt.Print("¿Deseas configurar un proyecto ahora? [S/n]: ")
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans == "" || ans == "s" || ans == "si" || ans == "sí" {
		if err := configureProject(reader, cfg); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println(clGreen + "✓ Listo!" + clReset + " Ya puedes usar: " + clBold + "gtt g -i APP-XXXX ..." + clReset)
	return nil
}

// configureProject runs the interactive project setup and saves to cfg.
func configureProject(reader *bufio.Reader, cfg *config.Config) error {
	fmt.Println()
	fmt.Println(clCyan + clBold + "── Configuración de proyecto ──" + clReset)

	name, err := prompt(reader, "Nombre del proyecto (ej: echo)")
	if err != nil {
		return err
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

	fmt.Printf(clGreen+"✓"+clReset+" Proyecto '%s' configurado\n", name)
	return nil
}

func prompt(r *bufio.Reader, label string) (string, error) {
	for {
		fmt.Printf("%s: ", label)
		val, err := r.ReadString('\n')
		if err != nil {
			return "", err
		}
		val = strings.TrimSpace(val)
		if val != "" {
			return val, nil
		}
		fmt.Println("  Este campo es requerido.")
	}
}

func promptWithDefault(r *bufio.Reader, label, defaultVal string) (string, error) {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	val, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultVal, nil
	}
	return val, nil
}

func promptOptional(r *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s (Enter para omitir): ", label)
	val, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(val), nil
}
