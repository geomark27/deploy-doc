package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/geomark27/deploy-doc/cmd"
	"github.com/geomark27/deploy-doc/internal/build"
	"github.com/geomark27/deploy-doc/internal/installer"
	"github.com/geomark27/deploy-doc/internal/updater"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	if !installer.IsInstalled() {
		fmt.Println("deploy-doc no está instalado en este sistema.")
		fmt.Printf("Destino: %s\n\n", installer.InstallDir())
		fmt.Println("La instalación realizará las siguientes acciones:")
		fmt.Println("  1. Copiar el ejecutable al directorio de destino")
		fmt.Println("  2. Agregar ese directorio al PATH del usuario")
		fmt.Println()
		fmt.Print("¿Deseas instalar deploy-doc ahora? [S/n]: ")

		ans, _ := reader.ReadString('\n')
		ans = strings.TrimSpace(strings.ToLower(ans))
		if ans != "" && ans != "s" && ans != "si" && ans != "sí" {
			fmt.Println("Instalación cancelada.")
			fmt.Printf("\nPuedes instalarlo manualmente copiando el ejecutable a una carpeta en tu PATH.\n")
			pause(reader)
			os.Exit(0)
		}

		fmt.Println()
		if err := installer.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error durante la instalacion: %v\n\n", err)
			fmt.Println("Puedes instalarlo manualmente copiando el ejecutable a una carpeta en tu PATH.")
			pause(reader)
			os.Exit(1)
		}

		fmt.Println("OK Binario copiado")
		fmt.Println("OK Agregado al PATH del usuario")
		fmt.Println()
		fmt.Println("----------------------------------------")
		fmt.Println("  deploy-doc instalado correctamente!")
		fmt.Println("----------------------------------------")
		fmt.Println()

		if runtime.GOOS == "windows" {
			fmt.Println("  Cierra y vuelve a abrir la terminal, luego ejecuta:")
		} else {
			fmt.Println("  Abre una nueva terminal o ejecuta:")
			fmt.Println("    source ~/.zshrc   (zsh)")
			fmt.Println("    source ~/.bashrc  (bash)")
			fmt.Println()
			fmt.Println("  Luego ejecuta:")
		}

		fmt.Println()
		fmt.Println("    deploy-doc init")
		fmt.Println()
		pause(reader)
		os.Exit(0)
	}

	// Clean up leftover .old binary on Windows from a previous update
	updater.CleanOldBinary()

	// Background update check (skip on update/help/version commands)
	updateCh := make(chan string, 1)
	if shouldCheckUpdate() {
		go func() {
			latest, err := updater.CheckLatest(build.Version)
			if err == nil && latest != "" {
				updateCh <- latest
			}
		}()
	}

	err := cmd.Execute()

	// Print update notification after command finishes
	select {
	case latest := <-updateCh:
		fmt.Printf("\nNueva version disponible: %s  →  ejecuta: deploy-doc update\n", latest)
	default:
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
}

func shouldCheckUpdate() bool {
	if len(os.Args) < 2 {
		return false
	}
	switch os.Args[1] {
	case "update", "version", "--version", "-v", "help", "--help", "-h":
		return false
	}
	return true
}

func pause(r *bufio.Reader) {
	fmt.Print("\nPresiona Enter para cerrar...")
	r.ReadString('\n') //nolint
}
