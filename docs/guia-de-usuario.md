# Guía de Usuario — deploy-doc

`deploy-doc` es una herramienta de línea de comandos que genera automáticamente documentos de despliegue en Confluence a partir de un issue de Jira y los commits de Git correspondientes.

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
| Linux / macOS | `~/.local/bin/deploy-doc` |
| Windows | `%LOCALAPPDATA%\Programs\deploy-doc\deploy-doc.exe` |

**Linux / macOS**

```bash
chmod +x deploy-doc-linux   # o deploy-doc-macos
./deploy-doc-linux
```

Al terminar la instalación, recarga tu shell:

```bash
source ~/.zshrc   # si usas zsh
source ~/.bashrc  # si usas bash
```

**Windows**

Ejecuta `deploy-doc-windows.exe` desde PowerShell o haciendo doble clic. Después de la instalación, cierra y vuelve a abrir PowerShell.

Verifica que quedó instalado:

```bash
deploy-doc version
```

---

## Configuración

Antes de usar `generate`, debes guardar tus credenciales de Atlassian con el comando `init`.

```bash
deploy-doc init
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

Un **proyecto** es una configuración con nombre que asocia las rutas locales de tus repositorios (backend y/o frontend) al CLI. Esto permite ejecutar `deploy-doc generate` desde **cualquier carpeta** sin tener que estar parado dentro del repositorio correspondiente.

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
deploy-doc init
```

No recibe flags. Es un asistente interactivo que guía campo por campo. Si ya existe una configuración guardada, pregunta si deseas sobreescribirla (`[s/N]`, por defecto No). Los proyectos configurados se conservan siempre.

---

### generate

Genera un documento de despliegue en Confluence.

```bash
deploy-doc generate --issue <ISSUE_KEY> [--commit-backend <HASH>] [--commit-frontend <HASH>] [--project <NOMBRE>]
```

**Flags disponibles:**

| Flag | Requerido | Descripción |
|---|---|---|
| `--issue` | Sí | Clave del issue en Jira (ej: `APP-1999`) |
| `--commit-backend` | Condicional* | Hash(es) del commit de backend. Acepta varios separados por coma |
| `--commit-frontend` | Condicional* | Hash(es) del commit de frontend. Acepta varios separados por coma |
| `--project` | No | Nombre del proyecto a usar. Si se omite, usa el `default_project` |
| `--dry-run` | No | Imprime el documento ADF en stdout sin crear nada en Confluence |

> *Al menos uno de los dos commits es requerido. Los flags aceptan tanto `--flag valor` como `--flag=valor`.

**Comportamiento con proyectos:**

- Si el proyecto tiene `backend_path`, git buscará el commit en esa ruta.
- Si el proyecto no tiene `backend_path` (o no hay proyecto configurado), git buscará en el directorio actual.
- Si se pasa `--commit-frontend` pero el proyecto no tiene `frontend_path`, se mostrará una advertencia y git usará el directorio actual.
- Si se especifica `--project noexiste`, el comando falla con un mensaje claro.

**Ejemplos:**

```bash
# Usando el proyecto por defecto
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86

# Especificando proyecto
deploy-doc generate --project ecuapass --issue ECU-123 --commit-backend abc1234

# Frontend y backend con proyecto por defecto
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86 --commit-frontend 5bd0cea0

# Múltiples commits (separados por coma)
deploy-doc generate --issue APP-1999 --commit-backend abc123,def456

# Vista previa del documento sin publicar en Confluence
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86 --dry-run

# Sin proyectos configurados (git corre en el directorio actual)
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86
```

**Flujo interactivo del comando:**

1. Muestra el proyecto activo (si hay uno configurado)
2. Busca el issue en Jira y muestra su título
3. Verifica si ya existe un documento de despliegue para ese issue
   - Si existe, pregunta qué hacer:
     ```
     [1] Actualizar el documento existente
     [2] Crear uno nuevo de todas formas
     [3] Cancelar
     ```
4. Lee los archivos modificados en cada commit (usando las rutas del proyecto si están configuradas). Soporta múltiples commits por servicio
5. Si un commit falla pero el otro tiene éxito, continúa con la información disponible
6. Muestra tus últimos documentos de despliegue para elegir la ubicación en Confluence (hasta 10 opciones)
7. Pide confirmación antes de crear o actualizar
8. Crea o actualiza la página y muestra la URL resultante

---

### project

Gestiona los proyectos configurados en el CLI.

```bash
deploy-doc project <subcomando> [opciones]
```

#### project list

Lista todos los proyectos configurados. El proyecto por defecto se marca con `*`.

```bash
deploy-doc project list
```

Salida ejemplo:
```
PROYECTO        BACKEND PATH                              FRONTEND PATH
--------------------------------------------------------------------
* echo          C:\laragon\www\operativo-api              C:\Proyects\Angular\echo-logistics
  ecuapass      C:\laragon\www\ecuapass-api               (no configurado)

* Proyecto por defecto: echo
```

#### project add

Asistente interactivo para agregar un nuevo proyecto.

```bash
deploy-doc project add
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
deploy-doc project default <nombre>
```

Ejemplo:
```bash
deploy-doc project default ecuapass
```

#### project remove

Elimina un proyecto. Pide confirmación antes de eliminar.

```bash
deploy-doc project remove <nombre>
```

Si el proyecto eliminado era el por defecto, avisa y limpia la configuración de default.

---

### update

Verifica si hay una nueva versión disponible y actualiza el CLI automáticamente.

```bash
deploy-doc update
```

Descarga el binario de la última versión desde GitHub Releases y reemplaza el ejecutable actual. En Windows, reinicia la terminal después de actualizar.

> **Nota:** Al usar cualquier comando (`generate`, `init`, etc.), el CLI verifica automáticamente si hay una versión nueva disponible y te notifica al finalizar:
> ```
> Nueva version disponible: v1.1.0  →  ejecuta: deploy-doc update
> ```

---

### version

Muestra la versión instalada del CLI.

```bash
deploy-doc version
```

---

### help

Muestra la ayuda general con los comandos disponibles.

```bash
deploy-doc help
deploy-doc --help
deploy-doc -h
```

---

## Flujo de trabajo típico

### Primera vez

```bash
# 1. Instala el binario (ejecutar el .exe descargado)
# 2. Configura credenciales y primer proyecto
deploy-doc init

# 3. Verifica la configuración
deploy-doc project list
```

### Uso diario

```bash
# 1. Obtén el hash de tu commit (en cualquier terminal, no importa el directorio)
#    Si tienes el proyecto configurado, no necesitas estar en el repo

# 2. Genera el documento
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86

# 3. Si trabajas en otro proyecto
deploy-doc generate --project ecuapass --issue ECU-200 --commit-backend abc1234
```

### Ejemplo multi-proyecto

```bash
# Tienes configurado:
# - echo (default): operativo-api + echo-logistics
# - ecuapass: ecuapass-api

# Generar doc de echo (usa el default)
deploy-doc generate --issue APP-1999 \
  --commit-backend 27cefd8671946ab5a617688a6933777b234ebef6 \
  --commit-frontend 5bd0cea0d5033eec0ad74ba302bee81fcc194730

# Generar doc de ecuapass (especifica el proyecto)
deploy-doc generate --project ecuapass \
  --issue ECU-123 \
  --commit-backend a3f8c12d9e4b56f10987654321abcdef01234567
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
| `configuración incompleta. Corre: deploy-doc init` | No hay credenciales guardadas | Ejecuta `deploy-doc init` |
| `credenciales inválidas (401)` | Token vencido o incorrecto | Genera un nuevo token y corre `deploy-doc init` |
| `sin permisos (403)` | El token no tiene acceso al recurso | Verifica los permisos del token en Atlassian |
| `recurso no encontrado (404)` | El issue no existe o está mal escrito | Verifica la clave del issue en Jira |
| `error al leer el commit <hash>` | El hash no existe en el repo configurado | Verifica el hash con `git log` desde el repositorio correcto |
| `git no encontrado en el sistema` | Git no está instalado o no está en PATH | Instala Git o agrégalo al PATH |
| `no se encontraron documentos de despliegue previos` | No hay páginas previas como referencia de ubicación | Crea un documento manualmente en Confluence como base |
| `proyecto 'X' no encontrado` | El proyecto no está configurado | Usa `deploy-doc project list` para ver los proyectos disponibles o `deploy-doc project add` para agregar uno |
| `⚠ Advertencia: el proyecto no tiene backend_path configurado` | El proyecto existe pero no tiene ruta de backend | Configura la ruta con `deploy-doc project add` sobreescribiendo el proyecto existente |

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

deploy-doc generate --issue APP-1999 --commit-backend $COMMIT_SHA
```

> Las rutas de proyectos no se pueden definir como variables de entorno; deben configurarse en el archivo `config.yaml`.

---

## Desinstalación manual

**Linux / macOS:**

```bash
rm ~/.local/bin/deploy-doc
rm -rf ~/.config/deploy-doc
```

**Windows (PowerShell):**

```powershell
Remove-Item "$env:LOCALAPPDATA\Programs\deploy-doc" -Recurse
Remove-Item "$env:USERPROFILE\.config\deploy-doc" -Recurse
```
