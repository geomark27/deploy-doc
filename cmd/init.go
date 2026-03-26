package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/geomark27/deploy-doc/internal/config"
)

func runInit(args []string) error {
	fmt.Println("=== Configuración de deploy-doc ===")
	fmt.Println("Tus credenciales se guardarán en ~/.config/deploy-doc/config.yaml")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	email, err := prompt(reader, "Atlassian email")
	if err != nil {
		return err
	}

	token, err := prompt(reader, "Atlassian API token (https://id.atlassian.com/manage-profile/security/api-tokens)")
	if err != nil {
		return err
	}

	baseURL, err := promptWithDefault(reader, "Base URL", "https://torresytorres.atlassian.net")
	if err != nil {
		return err
	}

	cfg := &config.Config{
		AtlassianEmail: email,
		AtlassianToken: token,
		BaseURL:        baseURL,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error guardando configuración: %w", err)
	}

	path, _ := config.ConfigPath()
	fmt.Printf("\n✓ Configuración guardada en %s\n", path)
	fmt.Println("Ya puedes usar: deploy-doc generate --issue APP-XXXX ...")
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