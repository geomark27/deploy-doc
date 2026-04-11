# Arquitectura técnica — gtt

> Para desarrolladores y agentes IA. Describe el diseño interno del proyecto.

---

## Propósito

CLI en Go que automatiza la creación de documentos de despliegue en Confluence. Orquesta tres fuentes: **Jira** (título del issue), **Git** (archivos modificados en un commit), y **Confluence** (creación/actualización de la página en formato ADF).

El binario se llama `gtt`. El módulo Go y el repo de GitHub mantienen el nombre original `deploy-doc` — solo el ejecutable distribuido cambió.

---

## Estructura de paquetes

```
deploy-doc/
├── main.go                        # Entry point: self-installer → update check → cmd.Execute()
├── cmd/
│   ├── root.go                    # Router + constantes de color ANSI + helpers stepLabel/okLine/warnLine/errLine
│   ├── init.go                    # Comando: gtt init (interactivo)
│   ├── generate.go                # Comando: gtt g|gen|generate (flujo principal)
│   ├── project.go                 # Comando: gtt project (list|ls/add/default/remove)
│   └── update.go                  # Comando: gtt update (auto-actualización)
├── internal/
│   ├── build/
│   │   └── version.go             # var Version = "dev" (sobreescrita por ldflags)
│   ├── config/
│   │   └── config.go              # Config + ProjectConfig structs, Load/Save con yaml.v3
│   ├── git/
│   │   └── git.go                 # GetChangedFiles, GetChangedFilesMulti, GroupByDirectory
│   ├── atlassian/
│   │   ├── client.go              # HTTP client base con Basic Auth
│   │   ├── jira.go                # GetIssue — Jira REST API v3
│   │   └── confluence.go          # FindLastDeployDoc, FindDeployDocByIssue, GetPage, CreatePage, UpdatePage
│   ├── document/
│   │   └── builder.go             # Build ADF como map[string]any desde DeployDoc struct
│   ├── installer/
│   │   ├── installer.go           # Self-install: copia el binario a ~/.local/bin
│   │   ├── path_unix.go           # Lógica de PATH para Linux/macOS
│   │   └── path_windows.go        # Lógica de PATH para Windows
│   └── updater/
│       └── updater.go             # CheckLatest, SelfUpdate desde GitHub Releases
```

---

## Flujo de datos — `gtt g`

```
flags (args)  — acepta -i/-b/-f/-p (cortos) y --issue/--commit-backend/etc. (largos)
    │
    ▼
config.Load()          env vars > ~/.config/deploy-doc/config.yaml
    │
    ├──► cfg.GetProject(name)       Resuelve proyecto (explícito > default > nil)
    │         └─► ProjectConfig{BackendPath, BackendRepo, FrontendPath, FrontendRepo, BitbucketOrg}
    │
    ├──► atlassian.GetIssue(key)    GET /rest/api/3/issue/{key}
    │
    ├──► atlassian.FindDeployDocByIssue()   GET /wiki/rest/api/search (CQL)
    │
    ├──► git.GetChangedFilesMulti(hashes, workDir)   git show --name-only
    │         └─► git.GroupByDirectory()             map[dir][]filename
    │
    ├──► document.Build(DeployDoc{...})    → map[string]any (ADF)
    │
    │    [si --dry-run: json.Encode → stdout y retorna]
    │
    └──► atlassian.CreatePage() / UpdatePage()   POST|PUT /wiki/api/v2/pages
```

---

## Diseño de configuración

**Structs:**
```go
type Config struct {
    AtlassianEmail string                    `yaml:"atlassian_email"`
    AtlassianToken string                    `yaml:"atlassian_token"`
    BaseURL        string                    `yaml:"base_url"`
    DefaultProject string                    `yaml:"default_project,omitempty"`
    Projects       map[string]*ProjectConfig `yaml:"projects,omitempty"`
}

type ProjectConfig struct {
    BackendPath  string `yaml:"backend_path,omitempty"`
    BackendRepo  string `yaml:"backend_repo,omitempty"`
    FrontendPath string `yaml:"frontend_path,omitempty"`
    FrontendRepo string `yaml:"frontend_repo,omitempty"`
    BitbucketOrg string `yaml:"bitbucket_org,omitempty"`
}
```

**Prioridad de carga:** env vars (`ATLASSIAN_EMAIL`, `ATLASSIAN_TOKEN`, `ATLASSIAN_BASE_URL`) → archivo YAML.

**Resolución de proyecto en `generate`:** flag `--project` / `-p` > `DefaultProject` > defaults hardcodeados (`operativo-api` / `echo-logistics` / `devtyt`).

**Formato del archivo:** YAML via `gopkg.in/yaml.v3`. Ruta: `~/.config/deploy-doc/config.yaml`, permisos `0600`.

---

## Formato de documento ADF

ADF (Atlassian Document Format) es el formato nativo de Confluence. Se construye como `map[string]any` en Go y se serializa a JSON string enviado como `body.value` con `representation: "atlas_doc_format"`.

**Estructura del documento generado:**

```
doc
├── table (headerTable)          Épica + Tarea(s) con inlineCard al issue Jira
├── heading(2) "Arquitecturas e interfaces"
│   ├── heading(3) "...Frontend:" (solo si hay archivos frontend)
│   │   └── table (filesTable)   Servidor | App web | Ubicación | Archivo | Observación
│   └── heading(3) "...Backend:" (solo si hay archivos backend)
│       └── table (filesTable)
└── heading(2) "A considerar:"
    └── table
        └── taskList             3 tareas predefinidas (TODO)
```

Los enlaces de "Observación" apuntan a `https://bitbucket.org/{org}/{repo}/commits/{hash}#chg-{filePath}`.

---

## APIs de Atlassian usadas

| Operación | Endpoint |
|---|---|
| Obtener issue Jira | `GET /rest/api/3/issue/{key}?fields=summary` |
| Buscar docs previos | `GET /wiki/rest/api/search?cql=...&limit=10` |
| Obtener página por ID | `GET /wiki/api/v2/pages/{id}` |
| Crear página | `POST /wiki/api/v2/pages` |
| Actualizar página | `PUT /wiki/api/v2/pages/{id}` |

Autenticación: Basic Auth con `base64(email:token)`.

---

## Versioning y release

La versión se embebe en el binario vía ldflags:

```
-X github.com/geomark27/deploy-doc/internal/build.Version=vX.Y.Z
```

Variable: `internal/build.Version`, fallback `"dev"` en `make run`.

`make build` / `make install` usan `git describe --tags --abbrev=0` para obtener la versión automáticamente. `make build-all` (usado por `make release`) recibe el tag nuevo como `VER=$$NEW_TAG`.

Targets de release: `make release` (patch +1), `make release-minor`, `make release-major`. Cada uno: lint → `build-all` con VER → git commit + tag → push → `gh release create` con assets `gtt-linux-amd64`, `gtt-windows-amd64.exe`, `gtt-darwin-amd64`.

### Semver comparison en `CheckLatest`

`updater.isNewer(candidate, base string) bool` parsea `vMAJOR.MINOR.PATCH` y compara numéricamente. `CheckLatest` solo retorna una versión si es **estrictamente mayor** que la actual — nunca ofrece downgrade.

---

## Convenciones de código

- Solo `gopkg.in/yaml.v3` y `golang.org/x/sys` como dependencias externas.
- **Texto user-facing en español** (mensajes, prompts, errores).
- **Manejo de errores:** siempre `fmt.Errorf("contexto: %w", err)`.
- **Router manual** en `cmd/root.go` — no se usa Cobra.
- **Colores ANSI** definidos como constantes en `cmd/root.go` y accesibles por todo el package `cmd`.
- No hay tests automatizados. Herramienta interna de equipo pequeño.
