package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/geomark27/deploy-doc/cmd"
	"github.com/geomark27/deploy-doc/internal/installer"
)

func main() {
	if !installer.IsInstalled() {
		fmt.Println("🚀 Instalando deploy-doc...")
		fmt.Printf("   Destino: %s\n\n", installer.InstallDir())

		if err := installer.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error durante la instalación: %v\n\n", err)
			fmt.Println("Puedes instalarlo manualmente copiando el ejecutable a una carpeta en tu PATH.")
			os.Exit(1)
		}

		fmt.Println("✓ Binario copiado")
		fmt.Println("✓ Agregado al PATH del usuario")
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  deploy-doc instalado correctamente!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		if runtime.GOOS == "windows" {
			fmt.Println("  Cierra y vuelve a abrir tu terminal, luego ejecuta:")
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
		os.Exit(0)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
