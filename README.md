# gtt

CLI para generar documentos de despliegue en Confluence automáticamente,
basándose en commits de Bitbucket e issues de Jira.

## Requisitos

- Git instalado en el sistema
- Acceso a Atlassian (Jira + Confluence)
- Token personal de Atlassian

---

## Instalación

### Opción A — Descargar el binario (recomendado)

1. Ve a [Releases](https://github.com/geomark27/deploy-doc/releases)
2. Descarga el binario según tu sistema operativo:
   - Windows → `gtt-windows-amd64.exe`
   - Linux   → `gtt-linux-amd64`
   - Mac     → `gtt-darwin-amd64`
3. Ejecútalo una vez — el instalador lo configura automáticamente en tu PATH
4. Configura tus credenciales:

```bash
gtt init
```

### Opción B — Compilar desde el código fuente

```bash
git clone https://github.com/geomark27/deploy-doc.git
cd deploy-doc
make install
```

> **Usuarios con versión anterior (deploy-doc):** ejecuta `deploy-doc update`
> y la migración a `gtt` ocurre automáticamente. Reinicia tu terminal al terminar.

---

## Configuración inicial

```bash
gtt init
```

Te pedirá:

| Campo | Descripción |
|---|---|
| Atlassian email | Tu correo corporativo |
| Atlassian API token | Genera uno en https://id.atlassian.com/manage-profile/security/api-tokens |

Las credenciales se guardan en `~/.config/deploy-doc/config.yaml` con permisos
de solo lectura para tu usuario. Nunca se commitean al repositorio.

También puedes configurarlo con variables de entorno:

```bash
export ATLASSIAN_EMAIL=tu@email.com
export ATLASSIAN_TOKEN=tu_token
export ATLASSIAN_BASE_URL=https://tuempresa.atlassian.net
export CONFLUENCE_SPACE_KEY=PA
```

### Claves que se editan directamente en el YAML

Estas no las pide `gtt init` porque solo aplican al comando `qa`:

```yaml
qa_email: "qa@tuempresa.com"   # habilita 'gtt qa' para ese usuario
qa_report:                     # encabezado del consolidado de pruebas
  lider_tecnico: "..."
  pmo: "..."
  qa: "..."

# Pasos de la sección "A considerar" de un documento nuevo.
deploy_checklist:
  - "Pasar backend al servidor"
  - "Ejecutar php artisan migrate"
  - "Pasar frontend"
```

Las claves de `qa_report` que falten se publican como `—`. Son datos de la
organización y por diseño no viven en el código — ver
[`docs/security/patrones-seguros.md`](docs/security/patrones-seguros.md) (P-008).

`deploy_checklist` se puede definir también **por proyecto** (gana sobre la
global), útil cuando los stacks difieren:

```yaml
projects:
  echo:
    backend_path: /home/tu/repos/echo-api
    deploy_checklist:
      - "Pasar API"
      - "npm run build && pasar dist"
```

Sin configurar, se usa un default genérico (`Pasar backend al servidor`,
`Ejecutar migraciones`, `Pasar frontend`). Solo aplica al **crear**: al
actualizar, la sección se preserva tal como quedó en Confluence.

---

## Uso

### Generar un documento de despliegue

```bash
# Forma corta (recomendada)
gtt g -i APP-1999 -b abc1234
gtt g -i APP-1999 -b abc1234 -f def5678

# Con proyecto específico
gtt g -p ecuapass -i ECU-123 -b abc1234

# También funciona la forma larga
gtt generate --issue APP-1999 --commit-backend abc1234
```

### Flags disponibles

| Corto | Largo | Requerido | Descripción |
|---|---|---|---|
| `-i` | `--issue` | ✓ | Key del issue en Jira (ej: APP-1999) |
| `-b` | `--commit-backend` | * | Hash(es) del commit de backend (separar varios con coma) |
| `-f` | `--commit-frontend` | * | Hash(es) del commit de frontend (separar varios con coma) |
| `-p` | `--project` | No | Proyecto a usar (usa el default si se omite) |
| — | `--dry-run` | No | Previsualiza el ADF sin publicar en Confluence |

*Al menos uno de los dos commits es requerido.

---

## Proyectos

Un proyecto asocia las rutas locales de tus repos con el CLI, permitiendo
ejecutar `gtt g` desde cualquier carpeta sin pararte dentro del repo.

```bash
gtt project add          # agregar proyecto (asistente interactivo)
gtt project list         # listar proyectos (alias: gtt project ls)
gtt project default echo # cambiar proyecto por defecto
gtt project remove echo  # eliminar proyecto
```

---

## Ejemplo completo

```bash
gtt g -i APP-1999 -b 27cefd86 -f 5bd0cea0
```

Output:

```
Proyecto: echo

[1/4] Buscando issue APP-1999...
      ✓ APP-1999 — Modal para Acción y Validaciones de Finalización de Operación

[2/4] Verificando documentos existentes...
      ✓ Ninguno encontrado

[3/4] Leyendo commits...
      ✓ backend  27cefd86  →  13 archivos
      ✓ frontend 5bd0cea0  →  15 archivos

[4/4] Seleccionando ubicación en Confluence...

  [1] Documento de Despliegue - APP-1998 - ...
  [2] Documento de Despliegue - APP-1996 - ...

  Opción (1-2): 1

Título: Documento de Despliegue - APP-1999 - Modal para Acción...

¿Confirmas la creación? [S/n]: S

      ✓ Documento creado!

  https://tuempresa.atlassian.net/wiki/spaces/PA/pages/...
```

---

## Actualización

```bash
gtt update
```

Descarga automáticamente la última versión desde GitHub y reemplaza el binario.
Solo actualiza si la versión remota es mayor a la instalada — nunca hace downgrade.

---

## Desarrollo

### Requisitos

- Go — la versión que declara `go.mod` (hoy `1.26.1`) o superior
- Make

### Comandos disponibles

```bash
make help            # Ver todos los comandos disponibles
make build           # Compilar para el OS actual
make run ARGS='...'  # Ejecutar sin compilar (dev mode)
make lint            # Formatear y analizar el código (fmt + vet)
make test            # Ejecutar los tests unitarios
make tidy            # Actualizar dependencias
make clean           # Limpiar binarios
make release         # Bump patch + compilar + push a GitHub
make release-minor   # Bump minor + push a GitHub
make release-major   # Bump major + push a GitHub
```

> `make lint` no corre los tests. Antes de un release: `make lint && make test`.

### Estructura del proyecto

```
deploy-doc/
├── cmd/
│   ├── root.go       # Router + constantes de color ANSI + help
│   ├── init.go       # gtt init
│   ├── generate.go   # gtt g / gen / generate
│   ├── qa.go         # gtt qa
│   ├── fetch.go      # gtt f / fetch
│   ├── project.go    # gtt project
│   └── update.go     # gtt update
├── internal/
│   ├── build/
│   │   └── version.go          # Variable Version (ldflags)
│   ├── config/
│   │   └── config.go           # Config + ProjectConfig + QAReportConfig
│   ├── git/
│   │   └── git.go              # GetChangedFilesMulti, GroupByDirectory
│   ├── atlassian/
│   │   ├── client.go           # Cliente HTTP con Basic Auth
│   │   ├── query.go            # quoteLiteral — escapado de JQL/CQL
│   │   ├── jira.go             # Jira REST API
│   │   ├── jira_qa.go          # JQL del consolidado QA
│   │   ├── confluence.go       # Confluence REST API (v1 + v2)
│   │   ├── confluence_fetch.go # Búsqueda y descarga de páginas
│   │   └── storage_text.go     # Storage format → texto plano
│   ├── document/
│   │   ├── builder.go          # ADF del documento de despliegue
│   │   ├── qa_builder.go       # ADF del consolidado QA
│   │   └── preserve.go         # Preservación de secciones editadas a mano
│   ├── installer/
│   │   └── installer.go        # Self-install al primer run
│   └── updater/
│       ├── updater.go          # CheckLatest + SelfUpdate (SHA-256)
│       └── notify.go           # Caché del aviso de versión
├── docs/
│   ├── arquitectura.md
│   ├── guia-de-usuario.md
│   ├── bitacora/               # Una entrada por versión
│   └── security/
│       └── patrones-seguros.md # Patrones P-001..P-008
├── main.go
├── Makefile
└── README.md
```

---

## Cómo obtener el hash de un commit

```bash
git log --oneline -1   # último commit
git log --oneline -5   # últimos 5
```

---

## Troubleshooting

**`configuración incompleta. Corre: gtt init`**
→ No se encontraron credenciales. Corre `gtt init` primero.

**`API error 401`**
→ Tu token de Atlassian es inválido o expiró. Genera uno nuevo en
https://id.atlassian.com/manage-profile/security/api-tokens y corre `gtt init`.

**`el commit XXX no tiene archivos o no existe`**
→ Verifica que el hash sea correcto y que el proyecto tenga configurado el path al repo.

**`no se encontraron documentos de despliegue previos`**
→ El CLI busca docs creados por tu usuario. Crea uno manualmente en
Confluence primero como referencia.

**`deploy-doc` sigue apareciendo después de migrar**
→ Reinicia la terminal. En Windows el archivo `deploy-doc.exe.old` se elimina
automáticamente la próxima vez que abras `gtt`.

---

## Licencia

MIT
