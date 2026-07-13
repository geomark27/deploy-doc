package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/geomark27/deploy-doc/cmd"
	"github.com/geomark27/deploy-doc/internal/build"
	"github.com/geomark27/deploy-doc/internal/config"
	"github.com/geomark27/deploy-doc/internal/installer"
	"github.com/geomark27/deploy-doc/internal/updater"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	if !installer.IsInstalled() {
		fmt.Println("gtt no está instalado en este sistema.")
		fmt.Printf("Destino: %s\n\n", installer.InstallDir())
		fmt.Println("La instalación realizará las siguientes acciones:")
		fmt.Println("  1. Copiar el ejecutable al directorio de destino")
		fmt.Println("  2. Agregar ese directorio al PATH del usuario")
		fmt.Println()
		fmt.Print("¿Deseas instalar gtt ahora? [S/n]: ")

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

		fmt.Println("✓ Binario copiado")
		fmt.Println("✓ Agregado al PATH del usuario")
		fmt.Println()
		fmt.Println("────────────────────────────────")
		fmt.Println("  gtt instalado correctamente!")
		fmt.Println("────────────────────────────────")
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
		fmt.Println("    gtt init")
		fmt.Println()
		pause(reader)
		os.Exit(0)
	}

	// Clean up leftover .old binary on Windows from a previous update
	updater.CleanOldBinary()

	// One-time migration: move ~/.config/deploy-doc/config.yaml → ~/.config/gtt/
	config.MigrateIfNeeded()

	// Update notification. The banner is served from a local cache (instant,
	// no network) and the GitHub API is only queried in the background once
	// per checkInterval. Skipped when stdout is not a terminal (pipes, files,
	// CI) so it never pollutes captured output; GTT_NO_UPDATE_CHECK forces off.
	doCheck := shouldCheckUpdate() && os.Getenv("GTT_NO_UPDATE_CHECK") == "" && isTerminal()
	notice := ""
	fresh := make(chan string, 1)
	if doCheck {
		notice = updater.CachedNotice(build.Version)
		if updater.NeedsRefresh() {
			go func() { fresh <- updater.Refresh(build.Version) }()
		} else {
			close(fresh)
		}
	}

	err := cmd.Execute()

	if doCheck {
		if notice != "" {
			// Cache already knew about a newer version — show it instantly.
			fmt.Println(notice)
		} else {
			// Nothing cached yet: briefly wait for the background check so the
			// very first run can still notify, without blocking on slow nets.
			select {
			case n := <-fresh:
				if n != "" {
					fmt.Println(n)
				}
			case <-time.After(1500 * time.Millisecond):
			}
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
}

// isTerminal reports whether stdout is an interactive terminal. When gtt's
// output is redirected to a file, a pipe, or a CI log, stdout is not a
// character device and the update banner is suppressed so it can't leak into
// captured output.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
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
