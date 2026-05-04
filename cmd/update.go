package cmd

import (
	"fmt"
	"runtime"

	"github.com/geomark27/deploy-doc/internal/build"
	"github.com/geomark27/deploy-doc/internal/updater"
)

func runUpdate(_ []string) error {
	fmt.Printf(clBold+"Versión actual : "+clReset+clCyan+"%s"+clReset+"\n", build.Version)
	fmt.Println("Buscando actualizaciones...")
	fmt.Println()

	latest, err := updater.CheckLatest(build.Version)
	if err != nil {
		return fmt.Errorf("error verificando version: %w", err)
	}

	if latest == "" {
		fmt.Println(clGreen + "✓" + clReset + " Ya tienes la versión más reciente.")
		return nil
	}

	fmt.Printf(clGreen+"Nueva versión disponible: "+clBold+"%s"+clReset+"\n\n", latest)

	if err := updater.SelfUpdate(latest); err != nil {
		return fmt.Errorf("error actualizando: %w", err)
	}

	fmt.Printf(clGreen+clBold+"✓ Actualizado a %s"+clReset+"\n", latest)
	if runtime.GOOS == "windows" {
		fmt.Println("\nReinicia tu terminal para aplicar los cambios.")
	}
	return nil
}
