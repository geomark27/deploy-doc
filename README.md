# deploy-doc

CLI para generar documentos de despliegue en Confluence automáticamente,
basándose en commits de Bitbucket e issues de Jira.

## Requisitos

- Git instalado en el sistema
- Acceso a Atlassian (Jira + Confluence)
- Token personal de Atlassian

## Instalación

### Opción A — Descargar el binario (recomendado)

1. Ve a [Releases](https://github.com/geomark27/deploy-doc/releases)
2. Descarga el binario según tu sistema operativo:
   - Windows → `deploy-doc-windows-amd64.exe`
   - Linux   → `deploy-doc-linux-amd64`
   - Mac     → `deploy-doc-darwin-amd64`
3. Configura tus credenciales:

```bash
deploy-doc init
```

### Opción B — Compilar desde el código fuente

```bash
git clone https://github.com/geomark27/deploy-doc.git
cd deploy-doc
make install
```

---

## Configuración inicial

Antes de usar el CLI por primera vez, corre:

```bash
deploy-doc init
```

Te pedirá:

| Campo | Descripción |
|---|---|
| Atlassian email | Tu correo corporativo |
| Atlassian API token | Genera uno en https://id.atlassian.com/manage-profile/security/api-tokens |
| Base URL | URL de tu instancia de Atlassian (ej: https://tuempresa.atlassian.net) |

Las credenciales se guardan en `~/.config/deploy-doc/config.yaml` y solo
son accesibles por tu usuario. Nunca se commitean al repositorio.

También puedes configurarlo con variables de entorno:

```bash
export ATLASSIAN_EMAIL=tu@email.com
export ATLASSIAN_TOKEN=tu_token
export ATLASSIAN_BASE_URL=https://tuempresa.atlassian.net
```

---

## Uso

### Generar un documento de despliegue

Párate en el repositorio donde está el commit y corre:

```bash
# Solo frontend
deploy-doc generate --issue APP-1999 --commit-frontend abc1234

# Solo backend
deploy-doc generate --issue APP-1999 --commit-backend abc1234

# Frontend y backend
deploy-doc generate --issue APP-1999 --commit-backend abc1234 --commit-frontend def5678
```

El CLI te mostrará tus últimos documentos de despliegue y te preguntará
en cuál ubicación crear el nuevo. Selecciona la más cercana al sprint actual.

### Flags disponibles

| Flag | Requerido | Descripción |
|---|---|---|
| `--issue` | ✓ | Key del issue en Jira (ej: APP-1999) |
| `--commit-backend` | * | Hash del commit de backend |
| `--commit-frontend` | * | Hash del commit de frontend |

*Al menos uno de los dos commits es requerido.

---

## Ejemplo completo

```bash
# Párate en el repo de frontend o backend
cd ~/proyectos/echo-logistics

# Genera el documento
deploy-doc generate \
  --issue APP-1999 \
  --commit-backend 27cefd8671946ab5a617688a6933777b234ebef6 \
  --commit-frontend 5bd0cea0d5033eec0ad74ba302bee81fcc194730
```

Output esperado:

```
Buscando issue APP-1999...
✓ APP-1999 - Modal para Acción y Validaciones de Finalización de Operación

Leyendo commit backend 27cefd86...
✓ 13 archivos encontrados

Leyendo commit frontend 5bd0cea0...
✓ 15 archivos encontrados

Buscando tus documentos de despliegue recientes...

¿Dónde deseas crear el documento? Selecciona una opción:

  [1] Documento de Despliegue - APP-1998 - ...
  [2] Documento de Despliegue - APP-1996 - ...

Opción (1-5): 1

Título: Documento de Despliegue - APP-1999 - Modal para Acción y Validaciones...

¿Confirmas la creación? [S/n]: S

Creando documento en Confluence...
✓ Documento creado exitosamente!
  https://tuempresa.atlassian.net/wiki/spaces/PA/pages/...
```

---

## Desarrollo

### Requisitos

- Go 1.26+
- Make

### Comandos disponibles

```bash
make help            # Ver todos los comandos disponibles
make build           # Compilar para el OS actual
make build-windows   # Compilar para Windows (.exe)
make run ARGS='...'  # Ejecutar sin compilar (dev mode)
make lint            # Formatear y analizar el código
make tidy            # Actualizar dependencias
make clean           # Limpiar binarios
make version         # Ver versión actual
make release         # Bump patch + compilar + push a GitHub
make release-minor   # Bump minor + push a GitHub
make release-major   # Bump major + push a GitHub
```

### Estructura del proyecto

```
deploy-doc/
├── cmd/
│   ├── root.go       # Entry point del CLI
│   ├── init.go       # Comando: deploy-doc init
│   └── generate.go   # Comando: deploy-doc generate
├── internal/
│   ├── config/
│   │   └── config.go     # Manejo de configuración
│   ├── git/
│   │   └── git.go        # Interacción con git
│   ├── atlassian/
│   │   ├── client.go     # Cliente HTTP base
│   │   ├── jira.go       # Jira REST API
│   │   └── confluence.go # Confluence REST API
│   └── document/
│       └── builder.go    # Construcción del ADF
├── main.go
├── Makefile
└── README.md
```

---

## Cómo obtener el hash de un commit

```bash
# Ver el último commit
git log --oneline -1

# Ver los últimos 5 commits
git log --oneline -5

# Ver commits de una rama específica
git log --oneline origin/APP-1999-mi-rama
```

---

## Troubleshooting

**`configuración incompleta. Corre: deploy-doc init`**
→ No se encontraron credenciales. Corre `deploy-doc init` primero.

**`API error 401`**
→ Tu token de Atlassian es inválido o expiró. Genera uno nuevo en
https://id.atlassian.com/manage-profile/security/api-tokens

**`el commit XXX no tiene archivos o no existe`**
→ Asegúrate de estar parado en el repositorio correcto al correr el comando.

**`no se encontraron documentos de despliegue previos`**
→ El CLI busca docs creados por tu usuario. Crea uno manualmente en
Confluence primero como referencia.

---

## Licencia

MIT
