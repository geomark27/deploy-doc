# Guía de Patrones Seguros de Desarrollo — gtt CLI

> **Documento vivo.** Cada vez que el equipo de seguridad o una revisión DevSecOps identifique un nuevo anti-patrón, debe agregarse aquí con su evidencia, solución y clasificación.
>
> **Última actualización:** 2026-05-04
> **Origen inicial:** Reporte DevSecOps v1.1.4 — `docs/issues/desarrollo15.html`

---

## Índice

| ID | Título | Severidad | Clasificación |
|----|--------|-----------|---------------|
| [P-001](#p-001) | No hardcodear URLs de instancia Atlassian | MEDIUM | OWASP A05 / T1071 |
| [P-002](#p-002) | No usar env vars como bypass de seguridad | HIGH | OWASP A07 / T1574 |
| [P-003](#p-003) | Valores de instancia Atlassian: config + flag de override | MEDIUM | OWASP A05 / T1071 |
| [P-004](#p-004) | Workspace y host del VCS: por proyecto + flag de override | MEDIUM | OWASP A05 / T1195 |
| [P-005](#p-005) | Resolver ruta absoluta antes de exec.Command | MEDIUM | OWASP A03 / T1059 |
| [P-006](#p-006) | Verificar integridad de binarios descargados | HIGH | OWASP A08 / T1105 |
| [P-007](#p-007) | Strings de ayuda en binario — falso positivo de scanner | INFO | T1552 |

---

## Contexto del proyecto

Esta herramienta no usa archivo `.env`. El único mecanismo de persistencia de configuración es `~/.config/deploy-doc/config.yaml`. Por eso el diseño de cada patrón debe contemplar **dos dimensiones**:

1. **Seguridad:** el valor no puede estar quemado en el código fuente ni en el binario.
2. **Practicidad:** el usuario no puede tener que editar el YAML cada vez que necesita usar un valor diferente.

La solución general para ambas dimensiones es un **sistema de prioridad por capas**:

```
Flag CLI (ej: --space ADN)     ← máxima prioridad, override puntual sin editar archivos
  └── ProjectConfig en YAML   ← default por proyecto (backends/frontends específicos)
        └── Config global      ← fallback general (config.yaml raíz)
              └── Valor por defecto en código (solo si aplica y no es dato corporativo)
```

Este patrón aparece en los CLIs de AWS, GitHub, Terraform y otros. Permite que:
- El usuario frecuente configure su default una sola vez.
- El usuario que alterna entre valores lo haga con un flag, sin tocar archivos.
- El código nunca hardcodee un dato que pertenece al entorno del cliente.

---

## P-001

### No hardcodear URLs de instancia Atlassian

**Severidad:** MEDIUM
**OWASP:** A05:2021-Security Misconfiguration
**MITRE:** T1071 — Application Layer Protocol

#### Por qué es un problema

Una URL corporativa embebida directamente en el código (`"https://empresa.atlassian.net"`) es detectada por motores EDR/XDR como Trend Micro Vision One como indicador de un agente C2. Adicionalmente, si el repositorio o binario son accesibles externamente, un atacante obtiene la topología interna sin ningún esfuerzo de reconocimiento. También ata el binario a una sola organización, eliminando la portabilidad.

#### Anti-patrón (lo que NO hacer)

```go
// cmd/init.go — URL corporativa quemada como default silencioso
cfg := &config.Config{
    BaseURL: "https://empresa.atlassian.net",  // ❌ hardcodeado
}
```

#### Patrón correcto

```go
// gtt init siempre pregunta la URL al usuario
baseURL, err := prompt(reader, "Atlassian base URL (ej: https://empresa.atlassian.net)")

cfg := &config.Config{
    BaseURL: baseURL,  // ✓ viene del usuario, se persiste en config.yaml
}
```

La URL queda guardada en `config.yaml` como `base_url` y se carga mediante `config.Load()` en cada ejecución. Nunca aparece como literal en el código.

#### Regla

> Ninguna URL de dominio corporativo, SaaS privado o intranet debe aparecer como literal en el código fuente. Siempre debe ser ingresada por el usuario en un wizard (`gtt init`) y persistida en `config.yaml`.

---

## P-002

### No usar variables de entorno como bypass de seguridad

**Severidad:** HIGH
**OWASP:** A07:2021-Identification and Authentication Failures
**MITRE:** T1574 — Hijack Execution Flow

#### Por qué es un problema

El problema **no** es usar variables de entorno (eso es una práctica válida). El problema es usar el *valor* de una env var para **deshabilitar controles de acceso o validaciones de seguridad**. Si `GTT_DEV=1` se establece accidentalmente en producción, se saltan verificaciones críticas. Los SIEM y XDR clasifican este patrón como T1574 — Conditional Backdoor.

#### Anti-patrón (lo que NO hacer)

```go
// cmd/qa.go — la env var desactiva la verificación de acceso completa
if os.Getenv("GTT_DEV") != "1" {       // ❌ backdoor condicional
    if cfg.QAEmail == "" || !strings.EqualFold(me.EmailAddress, cfg.QAEmail) {
        return fmt.Errorf("acceso denegado")
    }
}
// Con GTT_DEV=1 gtt qa cualquier usuario pasa la verificación
```

#### Patrón correcto

```go
// La verificación es siempre activa, sin importar el entorno
if cfg.QAEmail == "" || !strings.EqualFold(me.EmailAddress, cfg.QAEmail) {
    return fmt.Errorf("acceso denegado: este comando es exclusivo para el usuario QA configurado en qa_email")
}
// ✓ En desarrollo: agregar el email del dev en qa_email dentro de config.yaml
```

#### Distinción importante: env vars válidas vs. no válidas

| Uso | Válido | Ejemplo |
|-----|--------|---------|
| Configuración no sensible | ✓ | `LOG_LEVEL=debug`, timeout, feature flags cosméticos |
| Credenciales (email, token, base_url) | ✓ | `ATLASSIAN_TOKEN=...` con prioridad sobre el YAML |
| Bypass de autenticación o autorización | ❌ | `GTT_DEV=1` que salta una verificación de acceso |
| Bypass de validación de integridad | ❌ | `SKIP_HASH_CHECK=1` que omite la verificación de un binario |

#### Regla

> Las env vars son válidas para configuración (credenciales, URLs, log levels). Nunca deben usarse para evadir autenticación, autorización o validación de integridad. Si en desarrollo se necesita acceso con un usuario diferente al de producción, se configura `qa_email` en `config.yaml` con el email del dev.

---

## P-003

### Valores de instancia Atlassian: config + flag de override

**Severidad:** MEDIUM
**OWASP:** A05:2021-Security Misconfiguration
**MITRE:** T1071 — Reconnaissance

#### Por qué es un problema

Los identificadores de espacios de Confluence (`space = "PA"`) embebidos en el código revelan la estructura organizacional interna. Permiten enumerar espacios y mapear equipos sin autenticación. Adicionalmente, hardcodear un solo space key impide que el mismo usuario trabaje en múltiples espacios sin editar archivos de configuración.

#### Anti-patrón (lo que NO hacer)

```go
// internal/atlassian/confluence.go — space key quemado en 3 lugares
cql := fmt.Sprintf(`... AND space = "PA"`, issueKey)                        // ❌ línea 107
path := fmt.Sprintf("/wiki/api/v2/pages?title=%s&space-key=PA", ...)        // ❌ línea 209
cql := fmt.Sprintf(`... AND space = "PA" ORDER BY created DESC`, module)    // ❌ línea 238
```

#### Por qué "solo moverlo al YAML" no es suficiente

Si el space key vive únicamente en `config.yaml` como un valor único (`confluence_space_key: PA`), el usuario que trabaja en los espacios `PA` y `ADN` en el mismo día tiene que editar el YAML cada vez. Esto es impracticable y lleva a que la gente hardcodee el valor "para no molestarse". El patrón correcto elimina esa fricción.

#### Patrón correcto — sistema de prioridad por capas

```
--space FLAG (puntual)  >  ProjectConfig.SpaceKey  >  Config.ConfluenceSpaceKey (global)
```

```go
// 1. Config global: default para quien siempre trabaja en el mismo espacio
type Config struct {
    ...
    ConfluenceSpaceKey string `yaml:"confluence_space_key,omitempty"`  // ✓
}

// 2. ProjectConfig: override por proyecto (si un proyecto vive en otro espacio)
type ProjectConfig struct {
    ...
    ConfluenceSpaceKey string `yaml:"confluence_space_key,omitempty"`  // ✓ opcional
}

// 3. Flag CLI: override puntual sin tocar archivos
// gtt g -i APP-1234 --space ADN
// gtt qa -s 42 -m DAI --space ADN

// 4. Resolución en tiempo de ejecución
func resolveSpaceKey(flagValue string, proj *config.ProjectConfig, cfg *config.Config) string {
    if flagValue != "" {
        return flagValue                        // --space flag tiene máxima prioridad
    }
    if proj != nil && proj.ConfluenceSpaceKey != "" {
        return proj.ConfluenceSpaceKey          // default del proyecto
    }
    return cfg.ConfluenceSpaceKey              // fallback global
}
```

**Uso cotidiano sin fricción:**

```bash
# Usuario que siempre trabaja en PA — no hace nada especial:
gtt g -i APP-1234 -b abc1234

# Usuario que hoy trabaja en ADN — sin editar el YAML:
gtt g -i ADN-567 -b def5678 --space ADN
gtt qa -s 42 -m Aforo --space ADN
```

#### Regla

> Los identificadores de instancia Atlassian (space key, project key, board ID) nunca deben ser literales en el código. Deben existir como: (1) campo en `Config` como default global, (2) campo opcional en `ProjectConfig` para override por proyecto, y (3) flag CLI para override puntual. Este sistema de tres capas elimina tanto el riesgo de seguridad como la fricción de uso.

---

## P-004

### Workspace y host del VCS: por proyecto + flag de override

**Severidad:** MEDIUM
**OWASP:** A05:2021-Security Misconfiguration
**MITRE:** T1195 — Supply Chain / T1071 — Reconnaissance

#### Por qué es un problema

El host y la organización del VCS (`https://bitbucket.org/devtyt/`) quemados en el código revelan la estructura de repositorios internos y la convención de nombrado. Esto facilita reconocimiento pasivo de activos y ataques de Supply Chain. Los links generados en los documentos de Confluence dependen de este valor: si el workspace cambia, todos los links históricos quedan rotos.

#### Qué genera este valor en el documento final

El valor hardcodeado produce **dos tipos de links** visibles en el documento de Confluence:

```go
// internal/document/builder.go:103
repoURL := fmt.Sprintf("https://bitbucket.org/devtyt/%s", repoName)  // ❌

// Link 1: nombre del repo como enlace clickeable en la tabla
serverCell := tableCell(69, linkNode(repoURL, repoName))
// → [operativo-api](https://bitbucket.org/devtyt/operativo-api)

// Link 2: cada archivo tiene un link al diff del commit
commitURL := fmt.Sprintf("%s/commits/%s#chg-%s", repoURL, commitHash, filePath)
// → https://bitbucket.org/devtyt/operativo-api/commits/abc1234#chg-src/main.go
```

#### Por qué los datos VCS pertenecen al ProjectConfig, no al Config global

A diferencia del space key de Confluence (que puede variar por comando puntual), el VCS host y org son atributos fijos de cada proyecto: el repo `operativo-api` siempre vive en `bitbucket.org/devtyt`, independientemente del sprint o módulo. Por eso el lugar natural es `ProjectConfig`, no el config global.

#### Patrón correcto — por proyecto con flag de override

```
--vcs-org FLAG (puntual)  >  ProjectConfig.VCSOrg / VCSHost  >  ningún default hardcodeado
```

```go
// ProjectConfig: el VCS host y org son atributos del proyecto
type ProjectConfig struct {
    BackendPath  string `yaml:"backend_path,omitempty"`
    BackendRepo  string `yaml:"backend_repo,omitempty"`
    FrontendPath string `yaml:"frontend_path,omitempty"`
    FrontendRepo string `yaml:"frontend_repo,omitempty"`
    VCSHost      string `yaml:"vcs_host,omitempty"`   // ✓ ej: "https://bitbucket.org"
    VCSOrg       string `yaml:"vcs_org,omitempty"`    // ✓ ej: "devtyt"
}

// DeployDoc recibe los valores resueltos, no los calcula internamente
type DeployDoc struct {
    ...
    VCSHost string  // pasado desde cmd/generate.go
    VCSOrg  string  // pasado desde cmd/generate.go
}

// builder.go ya no tiene conocimiento del VCS específico
repoURL := fmt.Sprintf("%s/%s/%s", doc.VCSHost, doc.VCSOrg, repoName)  // ✓
```

```yaml
# config.yaml — ejemplo de proyecto configurado correctamente
projects:
  echo:
    backend_path: /home/user/repos/operativo-api
    backend_repo: operativo-api
    frontend_path: /home/user/repos/echo-logistics
    frontend_repo: echo-logistics
    vcs_host: https://bitbucket.org   # ✓ sin hardcodear en código
    vcs_org: devtyt                   # ✓ sin hardcodear en código
```

**Caso de override puntual** (ej: probar con un fork en otro workspace):

```bash
gtt g -i APP-1234 -b abc1234 --vcs-org mi-fork-org
```

#### Regla

> El VCS host y la organización/workspace pertenecen a `ProjectConfig`, ya que son atributos del proyecto y no cambian entre comandos del mismo proyecto. `gtt init` y `gtt project add` deben pedirlos al configurar un proyecto. Se acepta un flag `--vcs-org` para overrides puntuales. El módulo `document/builder.go` no debe conocer ningún VCS específico: recibe los datos ya resueltos desde la capa de comandos.

---

## P-005

### Resolver ruta absoluta antes de exec.Command

**Severidad:** MEDIUM
**OWASP:** A03:2021-Injection
**MITRE:** T1059 / T1574.007 — PATH Hijacking

#### Por qué es un problema

Invocar ejecutables con solo el nombre relativo (`exec.Command("git", ...)`) es vulnerable a PATH Hijacking: un atacante que controle cualquier directorio en `PATH` antes del binario legítimo puede colocar un ejecutable malicioso con el mismo nombre. Los EDR lo detectan como *Execution via Untrusted Binary Path*.

#### Anti-patrón (lo que NO hacer)

```go
// internal/git/git.go
cmd := exec.Command("git", "show", "--name-only", ...)  // ❌ ruta relativa
```

#### Patrón correcto

```go
gitPath, err := exec.LookPath("git")  // resuelve a /usr/bin/git  ✓
if err != nil {
    return nil, fmt.Errorf("git no encontrado en PATH: %w", err)
}
cmd := exec.Command(gitPath, "show", "--name-only", ...)  // ✓
```

#### Regla

> Toda llamada a `exec.Command` con un ejecutable del sistema operativo debe usar `exec.LookPath` primero para obtener la ruta absoluta. El error de `LookPath` debe propagarse con un mensaje claro que indique al usuario que el binario no está instalado.

---

## P-006

### Verificar integridad de binarios descargados

**Severidad:** HIGH
**OWASP:** A08:2021-Software and Data Integrity Failures
**MITRE:** T1105 — Ingress Tool Transfer

#### Por qué es un problema

Descargar un binario con `http.Get(url)` sin verificar el hash es exactamente el comportamiento primario de un Dropper. Los motores XDR bloquean y ponen en cuarentena procesos que escriban ejecutables sin verificación de integridad. Un atacante con acceso Man-in-the-Middle puede reemplazar el binario incluso sobre HTTPS.

#### Anti-patrón (lo que NO hacer)

```go
// internal/updater/updater.go:100
resp, err := http.Get(downloadURL)  //nolint:noctx  ❌ sin contexto, sin verificación
data, _ := io.ReadAll(resp.Body)
os.WriteFile(destPath, data, 0755)  // ❌ ejecutable escrito sin validar hash
```

#### Patrón correcto

```go
// 1. Descargar el archivo de checksums del mismo release en GitHub
checksumsURL := fmt.Sprintf(
    "https://github.com/%s/releases/download/%s/checksums.txt", repo, version,
)
expectedHash, err := fetchExpectedHash(checksumsURL, assetName())
if err != nil {
    return false, fmt.Errorf("no se pudo obtener el checksum oficial: %w", err)
}

// 2. Descargar el binario
resp, err := http.Get(downloadURL)
data, _ := io.ReadAll(resp.Body)

// 3. Verificar ANTES de escribir
actual := sha256.Sum256(data)
if hex.EncodeToString(actual[:]) != expectedHash {
    return false, fmt.Errorf("verificación de integridad fallida: el hash no coincide con el publicado en el release")
}

// 4. Solo si el hash es válido, escribir el ejecutable
os.WriteFile(destPath, data, 0755)  // ✓
```

#### Impacto en el proceso de release

Este patrón requiere que `make release` publique un archivo `checksums.txt` en el GitHub release junto con los binarios. El formato estándar es:

```
sha256hash  gtt-linux-amd64
sha256hash  gtt-windows-amd64.exe
sha256hash  gtt-darwin-amd64
```

Esto se puede generar en el `Makefile` con `sha256sum` (Linux/Mac) o integrarse en el flujo de `gh release create`.

#### Regla

> Toda descarga de binario o archivo ejecutable debe ir seguida de verificación de hash SHA-256 contra un archivo de checksums publicado en el mismo release. El proceso de `make release` es responsable de generar y publicar ese archivo. No escribir el ejecutable al disco hasta que el hash sea validado.

---

## P-007

### Strings de ayuda en binario — falso positivo de scanner

**Severidad:** INFO (falso positivo confirmado)
**MITRE:** T1552 — Credentials in Files

#### Contexto

Los scanners de binarios buscan patrones como `token`, `password`, `secret` en los bytes del ejecutable compilado. En un CLI Go, estos strings aparecen legítimamente en los textos de ayuda del usuario (ej: `"Atlassian API token"`, `"--token"`).

#### Cómo distinguir falso positivo de positivo real

| Patrón encontrado | Evaluación |
|-------------------|-----------|
| `"Atlassian API token (https://...)"` | ✓ Falso positivo — texto de ayuda |
| `"--token"` | ✓ Falso positivo — nombre de flag |
| `"ATATT3xFfGF0..."` (34+ caracteres alfanuméricos) | ❌ Positivo real — token de Atlassian |
| `"ghp_abc123xyz..."` | ❌ Positivo real — token de GitHub |
| `"sk-prod-abc123..."` | ❌ Positivo real — API key |

#### Regla

> Nunca incluir secretos reales (tokens, passwords, API keys) en el código fuente ni en constantes. Los textos de ayuda que *mencionen* las palabras `token` o `password` son normales y no representan un riesgo. Cuando el scanner los reporte, documentar la excepción con evidencia del contexto (línea de código o string exacto encontrado) en el ticket de seguridad correspondiente.

---

## Cómo agregar un nuevo patrón

1. Copiar la plantilla de cualquier sección existente.
2. Asignar el siguiente ID correlativo (`P-008`, `P-009`, ...).
3. Completar: severidad, clasificaciones, descripción del riesgo, anti-patrón con código real del proyecto, patrón correcto, y la **Regla** en forma de oración accionable.
4. Agregar la entrada al índice al inicio del archivo.
5. Referenciar el reporte o incidente de origen en el cuerpo de la sección.
6. Si el patrón implica un cambio en el diseño del CLI (nueva flag, nuevo campo en Config), documentar también en qué capa del sistema de prioridades vive el valor.
