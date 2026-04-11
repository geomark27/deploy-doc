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

	migrated, err := updater.SelfUpdate(latest)
	if err != nil {
		return fmt.Errorf("error actualizando: %w", err)
	}

	if migrated {
		fmt.Println()
		fmt.Println(clGreen + clBold + "✓ Migración completada: deploy-doc → gtt" + clReset)
		fmt.Println()
		fmt.Println("  Versión instalada : " + clBold + latest + clReset)
		fmt.Println("  Nuevo comando     : " + clGreen + clBold + "gtt" + clReset)
		fmt.Println()
		fmt.Println("  Todos los comandos siguen funcionando igual:")
		fmt.Println("    " + clBold + "gtt g -i APP-1999 -b <hash>" + clReset)
		fmt.Println("    " + clBold + "gtt project list" + clReset)
		fmt.Println("    " + clBold + "gtt update" + clReset)
		fmt.Println()
		if runtime.GOOS == "windows" {
			fmt.Println("  " + clYellow + "Reinicia tu terminal" + clReset + " para que 'gtt' quede disponible.")
		} else {
			fmt.Println("  " + clYellow + "Abre una nueva terminal" + clReset + " o ejecuta: source ~/.zshrc")
		}
	} else {
		fmt.Printf(clGreen+clBold+"✓ Actualizado a %s"+clReset+"\n", latest)
		if runtime.GOOS == "windows" {
			fmt.Println("\nReinicia tu terminal para aplicar los cambios.")
		}
	}
	return nil
}
