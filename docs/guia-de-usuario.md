# Guía de Usuario — gtt

`gtt` es una herramienta de línea de comandos que genera automáticamente documentos de despliegue en Confluence a partir de un issue de Jira y los commits de Git correspondientes.

---

## Índice

- [Requisitos](#requisitos)
- [Instalación](#instalación)
- [Configuración](#configuración)
- [Proyectos](#proyectos)
- [Comandos](#comandos)
  - [init](#init)
  - [generate](#generate)
  - [project](#project)
  - [update](#update)
  - [version](#version)
  - [help](#help)
- [Flujo de trabajo típico](#flujo-de-trabajo-típico)
- [Documento generado](#documento-generado)
- [Manejo de errores comunes](#manejo-de-errores-comunes)
- [Variables de entorno](#variables-de-entorno)
- [Desinstalación manual](#desinstalación-manual)

---

## Requisitos

- **Git** instalado y disponible en el PATH del sistema
- Acceso a una instancia de **Atlassian Cloud** (Jira + Confluence)
- Un **API token** de Atlassian ([generarlo aquí](https://id.atlassian.com/manage-profile/security/api-tokens))

---

## Instalación

Descarga el binario correspondiente a tu sistema operativo desde los releases del repositorio y ejecútalo una vez. La primera vez que se ejecuta, el instalador copia el binario automáticamente al PATH del usuario.

| Sistema operativo | Destino de instalación |
|---|---|
| Linux / macOS | `~/.local/bin/gtt` |
| Windows | `%LOCALAPPDATA%\Programs\deploy-doc\gtt.exe` |

**Linux / macOS**

```bash
chmod +x gtt-linux-amd64   # o gtt-darwin-amd64
./gtt-linux-amd64
```

Al terminar la instalación, recarga tu shell:

```bash
source ~/.zshrc   # si usas zsh
source ~/.bashrc  # si usas bash
```

**Windows**

Ejecuta `gtt-windows-amd64.exe` desde PowerShell o haciendo doble clic. Después de la instalación, cierra y vuelve a abrir PowerShell.

Verifica que quedó instalado:

```bash
gtt version
```

---

## Configuración

Antes de usar `generate`, debes guardar tus credenciales de Atlassian con el comando `init`.

```bash
gtt init
```

Se te pedirá:

| Campo | Descripción |
|---|---|
| `Atlassian email` | El correo con el que ingresas a Atlassian |
| `Atlassian API token` | Token generado en tu perfil de seguridad |

> Si ya existe una configuración guardada, el comando preguntará si deseas sobreescribirla antes de continuar. Los proyectos configurados se conservan siempre.

Al finalizar, `init` te ofrecerá configurar tu primer **proyecto** (ver sección [Proyectos](#proyectos)).

Las credenciales se guardan en:

```
~/.config/deploy-doc/config.yaml
```

Con permisos `0600` (solo lectura/escritura para tu usuario). El archivo tiene este formato:

```yaml
atlassian_email: tu@correo.com
atlassian_token: tu_api_token
base_url: https://tuempresa.atlassian.net
default_project: echo
projects:
  echo:
    backend_path: C:\laragon\www\operativo-api
    backend_repo: operativo-api
    frontend_path: C:\Proyects\Angular\echo-logistics
    frontend_repo: echo-logistics
  ecuapass:
    backend_path: C:\laragon\www\ecuapass-api
    backend_repo: ecuapass-api
```

> Puedes editar este archivo directamente si necesitas actualizar algún valor.

---

## Proyectos

Un **proyecto** es una configuración con nombre que asocia las rutas locales de tus repositorios (backend y/o frontend) al CLI. Esto permite ejecutar `gtt g` desde **cualquier carpeta** sin tener que estar parado dentro del repositorio correspondiente.

### ¿Por qué usar proyectos?

Sin proyectos, `git show` se ejecuta en el directorio actual. Si el commit pertenece al repo de frontend pero estás en el directorio de backend, el comando fallará.

Con proyectos configurados, el CLI sabe exactamente dónde están tus repositorios y los usa automáticamente.

### Estructura de un proyecto

| Campo | Descripción |
|---|---|
| `backend_path` | Ruta absoluta al repositorio de backend en tu máquina |
| `backend_repo` | Nombre del repositorio en Bitbucket |
| `frontend_path` | Ruta absoluta al repositorio de frontend en tu máquina |
| `frontend_repo` | Nombre del repositorio en Bitbucket |

Un proyecto puede tener solo backend, solo frontend, o ambos.

### Proyecto por defecto

Si configuras un `default_project`, el CLI lo usará automáticamente cuando no se especifica `--project` en `generate`.

### Paths en Windows

Las rutas en Windows deben escribirse con backslashes o con barras normales, ambas funcionan:

```
C:\laragon\www\operativo-api
C:/laragon/www/operativo-api
```

---

## Comandos

### init

Configura o actualiza las credenciales de Atlassian y opcionalmente configura un proyecto.

```bash
gtt init
```

No recibe flags. Es un asistente interactivo que guía campo por campo. Si ya existe una configuración guardada, pregunta si deseas sobreescribirla (`[s/N]`, por defecto No). Los proyectos configurados se conservan siempre.

---

### generate

Genera un documento de despliegue en Confluence. Disponible con tres nombres equivalentes: `g` (recomendado), `gen`, `generate`.

```bash
gtt g -i <ISSUE_KEY> [-b <HASH>] [-f <HASH>] [-p <PROYECTO>]
```

**Flags disponibles:**

| Flag corto | Flag largo | Requerido | Descripción |
|---|---|---|---|
| `-i` | `--issue` | Sí | Clave del issue en Jira (ej: `APP-1999`) |
| `-b` | `--commit-backend` | Condicional* | Hash(es) del commit de backend. Acepta varios separados por coma |
| `-f` | `--commit-frontend` | Condicional* | Hash(es) del commit de frontend. Acepta varios separados por coma |
| `-p` | `--project` | No | Nombre del proyecto a usar. Si se omite, usa el `default_project` |
| — | `--dry-run` | No | Imprime el documento ADF en stdout sin crear nada en Confluence |

> *Al menos uno de los dos commits es requerido. Los flags aceptan `-i APP-1999`, `-i=APP-1999`, `--issue APP-1999` y `--issue=APP-1999`.

**Comportamiento con proyectos:**

- Si el proyecto tiene `backend_path`, git buscará el commit en esa ruta.
- Si el proyecto no tiene `backend_path` (o no hay proyecto configurado), git buscará en el directorio actual.
- Si se pasa `-f` pero el proyecto no tiene `frontend_path`, se mostrará una advertencia y git usará el directorio actual.
- Si se especifica `-p noexiste`, el comando falla con un mensaje claro.

**Ejemplos:**

```bash
# Forma corta recomendada — proyecto por defecto
gtt g -i APP-1999 -b 27cefd86

# Con frontend
gtt g -i APP-1999 -b 27cefd86 -f 5bd0cea0

# Especificando proyecto
gtt g -p ecuapass -i ECU-123 -b abc1234

# Múltiples commits (separados por coma)
gtt g -i APP-1999 -b abc123,def456

# Vista previa sin publicar en Confluence
gtt g -i APP-1999 -b 27cefd86 --dry-run

# Usando flags largos (también válido)
gtt generate --issue APP-1999 --commit-backend 27cefd86
```

**Flujo interactivo del comando:**

```
Proyecto: echo

[1/4] Buscando issue APP-1999...
      ✓ APP-1999 — Deploy módulo de pagos

[2/4] Verificando documentos existentes...
      ✓ Ninguno encontrado   (o ⚠ con opciones si ya existe)

[3/4] Leyendo commits...
      ✓ backend  27cefd86  →  12 archivos
      ✓ frontend 5bd0cea0  →  8 archivos

[4/4] Seleccionando ubicación en Confluence...
      [1] Deploy - Sprint 42
      [2] Deploy - Sprint 41

  Opción (1-2): _
```

Si ya existe un documento para el issue, el paso `[2/4]` ofrece:
```
[1] Actualizar    [2] Crear nuevo    [3] Cancelar
```

---

### project

Gestiona los proyectos configurados en el CLI.

```bash
gtt project <subcomando> [opciones]
```

#### project list / project ls

Lista todos los proyectos configurados. El proyecto por defecto se marca con `*` en verde.

```bash
gtt project list
gtt project ls   # alias equivalente
```

#### project add

Asistente interactivo para agregar un nuevo proyecto.

```bash
gtt project add
```

Te pedirá:
- Nombre del proyecto
- Ruta del repositorio backend (opcional)
- Nombre del repositorio backend en Bitbucket
- Ruta del repositorio frontend (opcional)
- Nombre del repositorio frontend en Bitbucket
- Si establecerlo como proyecto por defecto

Si el proyecto ya existe, pregunta si deseas sobreescribirlo.

#### project default

Cambia el proyecto por defecto.

```bash
gtt project default <nombre>
```

#### project remove

Elimina un proyecto. Pide confirmación antes de eliminar.

```bash
gtt project remove <nombre>
```

Si el proyecto eliminado era el por defecto, avisa y limpia la configuración de default.

---

### update

Verifica si hay una nueva versión disponible y actualiza el CLI automáticamente.

```bash
gtt update
```

Descarga el binario de la última versión desde GitHub Releases y reemplaza el ejecutable actual. Solo actualiza si la versión remota es **estrictamente mayor** que la instalada — nunca hace downgrade. En Windows, reinicia la terminal después de actualizar.

> **Nota:** Al usar cualquier comando (`g`, `init`, etc.), el CLI verifica automáticamente si hay una versión nueva disponible y te notifica al finalizar:
> ```
> Nueva versión disponible: v1.1.1  →  ejecuta: gtt update
> ```

---

### version

Muestra la versión instalada del CLI.

```bash
gtt version
```

---

### help

Muestra la ayuda general con los comandos disponibles.

```bash
gtt help
gtt --help
gtt -h
```

---

## Flujo de trabajo típico

### Primera vez

```bash
# 1. Instala el binario (ejecutar el archivo descargado)
# 2. Configura credenciales y primer proyecto
gtt init

# 3. Verifica la configuración
gtt project list
```

### Uso diario

```bash
# Genera el documento (desde cualquier directorio)
gtt g -i APP-1999 -b 27cefd86

# Si trabajas en otro proyecto
gtt g -p ecuapass -i ECU-200 -b abc1234
```

### Ejemplo multi-proyecto

```bash
# Tienes configurado:
# - echo (default): operativo-api + echo-logistics
# - ecuapass: ecuapass-api

# Generar doc de echo con backend + frontend (usa el default)
gtt g -i APP-1999 -b 27cefd8671946ab5a617688a6933777b234ebef6 -f 5bd0cea0d5033eec0ad74ba302bee81fcc194730

# Generar doc de ecuapass (especifica el proyecto)
gtt g -p ecuapass -i ECU-123 -b a3f8c12d9e4b56f10987654321abcdef01234567
```

---

## Documento generado

Cada documento creado en Confluence incluye:

- **Tabla de encabezado**: enlace al issue de Jira (Tarea)
- **Sección "Arquitecturas e interfaces"**: tabla con los archivos modificados, organizados por directorio, con enlaces directos al commit en Bitbucket

  | Columna | Contenido |
  |---|---|
  | Servidor | Enlace al repositorio |
  | Aplicación web | Nombre de la aplicación |
  | Ubicación | Directorio del archivo |
  | Nombre del archivo | Archivos modificados |
  | Observación | Links directos al diff en Bitbucket |

- **Sección "A considerar"**: lista de tareas predefinidas para el proceso de despliegue:
  - Ejecutar php artisan migrate
  - Pasar backend al servidor
  - Pasar frontend

---

## Manejo de errores comunes

| Error | Causa | Solución |
|---|---|---|
| `configuración incompleta. Corre: gtt init` | No hay credenciales guardadas | Ejecuta `gtt init` |
| `credenciales inválidas (401)` | Token vencido o incorrecto | Genera un nuevo token y corre `gtt init` |
| `sin permisos (403)` | El token no tiene acceso al recurso | Verifica los permisos del token en Atlassian |
| `recurso no encontrado (404)` | El issue no existe o está mal escrito | Verifica la clave del issue en Jira |
| `error al leer el commit <hash>` | El hash no existe en el repo configurado | Verifica el hash con `git log` desde el repositorio correcto |
| `git no encontrado en el sistema` | Git no está instalado o no está en PATH | Instala Git o agrégalo al PATH |
| `no se encontraron documentos de despliegue previos` | No hay páginas previas como referencia de ubicación | Crea un documento manualmente en Confluence como base |
| `proyecto 'X' no encontrado` | El proyecto no está configurado | Usa `gtt project list` para ver los proyectos disponibles o `gtt project add` para agregar uno |
| `⚠ el proyecto no tiene backend_path configurado` | El proyecto existe pero no tiene ruta de backend | Configura la ruta con `gtt project add` sobreescribiendo el proyecto existente |

---

## Variables de entorno

Como alternativa al archivo de configuración, puedes definir las credenciales como variables de entorno. Tienen prioridad sobre el archivo `config.yaml`.

| Variable | Equivalente en config |
|---|---|
| `ATLASSIAN_EMAIL` | `atlassian_email` |
| `ATLASSIAN_TOKEN` | `atlassian_token` |
| `ATLASSIAN_BASE_URL` | `base_url` |

Ejemplo de uso en CI/CD:

```bash
export ATLASSIAN_EMAIL="ci@empresa.com"
export ATLASSIAN_TOKEN="tu_token"
export ATLASSIAN_BASE_URL="https://tuempresa.atlassian.net"

gtt g -i APP-1999 -b $COMMIT_SHA
```

> Las rutas de proyectos no se pueden definir como variables de entorno; deben configurarse en el archivo `config.yaml`.

---

## Desinstalación manual

**Linux / macOS:**

```bash
rm ~/.local/bin/gtt
rm -rf ~/.config/deploy-doc
```

**Windows (PowerShell):**

```powershell
Remove-Item "$env:LOCALAPPDATA\Programs\deploy-doc" -Recurse
Remove-Item "$env:USERPROFILE\.config\deploy-doc" -Recurse
```
