# Guía de Usuario — deploy-doc

`deploy-doc` es una herramienta de línea de comandos que genera automáticamente documentos de despliegue en Confluence a partir de un issue de Jira y los commits de Git correspondientes.

---

## Índice

- [Requisitos](#requisitos)
- [Instalación](#instalación)
- [Configuración](#configuración)
- [Comandos](#comandos)
  - [init](#init)
  - [generate](#generate)
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

Ejecuta `deploy-doc-windows.exe` desde PowerShell. Después de la instalación, cierra y vuelve a abrir PowerShell.

Verifica que quedó instalado:

```bash
deploy-doc --help
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
| `Base URL` | URL base de tu instancia (por defecto: `https://torresytorres.atlassian.net`) |

Las credenciales se guardan en:

```
~/.config/deploy-doc/config.yaml
```

Con permisos `0600` (solo lectura/escritura para tu usuario). El archivo tiene este formato:

```yaml
atlassian_email: tu@correo.com
atlassian_token: tu_api_token
base_url: https://tuempresa.atlassian.net
```

> Puedes editar este archivo directamente si necesitas actualizar algún valor.

---

## Comandos

### init

Configura o actualiza las credenciales de Atlassian.

```
deploy-doc init
```

No recibe flags. Es un asistente interactivo que guía campo por campo.

---

### generate

Genera un documento de despliegue en Confluence.

```
deploy-doc generate --issue <ISSUE_KEY> [--commit-backend <HASH>] [--commit-frontend <HASH>]
```

**Flags disponibles:**

| Flag | Requerido | Descripción |
|---|---|---|
| `--issue` | Sí | Clave del issue en Jira (ej: `APP-1999`) |
| `--commit-backend` | Condicional* | Hash del commit en el repositorio backend |
| `--commit-frontend` | Condicional* | Hash del commit en el repositorio frontend |

> *Al menos uno de los dos commits es requerido.

**Ejemplos:**

Solo backend:
```bash
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86
```

Solo frontend:
```bash
deploy-doc generate --issue APP-2045 --commit-frontend 5bd0cea0
```

Backend y frontend:
```bash
deploy-doc generate --issue APP-1999 --commit-backend 27cefd86 --commit-frontend 5bd0cea0
```

**Flujo interactivo del comando:**

1. Busca el issue en Jira y muestra su título
2. Verifica si ya existe un documento de despliegue para ese issue
   - Si existe, pregunta qué hacer:
     ```
     [1] Actualizar el documento existente
     [2] Crear uno nuevo de todas formas
     [3] Cancelar
     ```
3. Lee los archivos modificados en cada commit
4. Muestra tus últimos documentos de despliegue para elegir la ubicación en Confluence
5. Pide confirmación antes de crear o actualizar
6. Crea o actualiza la página y muestra la URL resultante

---

### help

Muestra la ayuda general con los comandos disponibles.

```bash
deploy-doc help
deploy-doc --help
deploy-doc -h
```

Salida:
```
deploy-doc - Generador de documentos de despliegue

Uso:
  deploy-doc <comando> [opciones]

Comandos:
  init      Configura tus credenciales de Atlassian
  generate  Genera un documento de despliegue

Ejemplos:
  deploy-doc init
  deploy-doc generate --issue APP-1999 --commit-backend 27cefd86 --commit-frontend 5bd0cea0
```

---

## Flujo de trabajo típico

```
1. (Primera vez) Instala el binario y ejecuta deploy-doc init

2. Cuando termines un desarrollo, obtén el hash de tu commit:
   git log --oneline -5

3. Genera el documento:
   deploy-doc generate --issue APP-1999 --commit-backend a3f8c12

4. Selecciona la ubicación en Confluence (basándose en documentos previos)

5. Confirma → el documento queda creado y se muestra la URL
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
| `error al leer el commit <hash>: ...` | El hash no existe en el repo actual | Verifica el hash con `git log` desde el repositorio correcto |
| `git no encontrado en el sistema` | Git no está instalado o no está en PATH | Instala Git o agrégalo al PATH |
| `no se encontraron documentos de despliegue previos` | No hay páginas previas como referencia de ubicación | Crea un documento manualmente en Confluence como base |

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
Remove-Item "$env:APPDATA\.config\deploy-doc" -Recurse
```
