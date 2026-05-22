---
description: Scaffolding de un nuevo comando gtt. Crea cmd/<nombre>.go con la estructura estándar del proyecto (sin Cobra, router de mapa), lo registra en root.go, compila y valida seguridad. Uso: /new-command <nombre>
---

# /new-command

Crea el scaffolding de un nuevo comando CLI para `gtt` siguiendo la arquitectura del proyecto (sin Cobra, router basado en `map[string]func([]string)error`).

Uso: `/new-command <nombre>` — por ejemplo `/new-command sync` crea el comando `gtt sync`.

## Instrucciones

El argumento es: `$ARGUMENTS`

1. **Valida el argumento**: si está vacío, pregunta al usuario el nombre del comando antes de continuar.

2. **Lee estos archivos** para entender los patrones actuales del proyecto antes de generar código:
   - `cmd/root.go` — para ver cómo se registran los comandos en el map
   - `cmd/qa.go` — como referencia de un comando completo con flags, prompts y pasos
   - `internal/config/config.go` — para saber qué campos expone `Config`

3. **Crea `cmd/<nombre>.go`** con esta estructura:

```go
package cmd

import (
    "bufio"
    "fmt"
    "os"

    "github.com/geomark27/deploy-doc/internal/atlassian"
    "github.com/geomark27/deploy-doc/internal/config"
)

var <nombre>ShortFlags = map[string]string{
    // "-x": "--long-flag",
}

func run<Nombre>(args []string) error {
    flags := parseFlagsWithShorts(args, <nombre>ShortFlags)
    _ = flags

    reader := bufio.NewReader(os.Stdin)
    _ = reader

    cfg, err := config.Load()
    if err != nil {
        return err
    }
    client := atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)
    _ = client

    // TODO: implementar lógica del comando

    fmt.Println("gtt <nombre>: no implementado aún")
    return nil
}
```

4. **Registra el comando en `cmd/root.go`**: agrega una entrada al map de comandos:
```go
"<nombre>": run<Nombre>,
```

5. **Verifica que compile**: ejecuta `go build ./...` y corrige cualquier error.

6. **Ejecuta `/security-check cmd/<nombre>.go`** sobre el archivo recién creado para confirmar que no introduce hallazgos.

7. Muestra al usuario:
   - Los archivos creados/modificados
   - Cómo invocar el nuevo comando: `gtt <nombre>`
   - Un recordatorio de los pasos pendientes de implementación (TODO en el archivo)
