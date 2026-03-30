package cmd

import (
	"fmt"
	"runtime"

	"github.com/geomark27/deploy-doc/internal/build"
	"github.com/geomark27/deploy-doc/internal/updater"
)

func runUpdate(_ []string) error {
	fmt.Printf("Version actual : %s\n", build.Version)
	fmt.Println("Buscando actualizaciones...")
	fmt.Println()

	latest, err := updater.CheckLatest(build.Version)
	if err != nil {
		return fmt.Errorf("error verificando version: %w", err)
	}

	if latest == "" {
		fmt.Println("Ya tienes la version mas reciente.")
		return nil
	}

	fmt.Printf("Nueva version disponible: %s\n\n", latest)

	if err := updater.SelfUpdate(latest); err != nil {
		return fmt.Errorf("error actualizando: %w", err)
	}

	fmt.Printf("OK Actualizado a %s\n", latest)
	if runtime.GOOS == "windows" {
		fmt.Println("\nReinicia tu terminal para aplicar los cambios.")
	}
	return nil
}
