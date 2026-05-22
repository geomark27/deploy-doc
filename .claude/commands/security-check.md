---
description: Revisa el código modificado contra los 7 patrones de seguridad del proyecto (P-001 a P-007). Sin argumentos evalúa los archivos cambiados en git; con argumentos evalúa solo esos archivos.
---

# /security-check

Revisa el código modificado o nuevo contra los 7 patrones de seguridad definidos en `docs/security/patrones-seguros.md`.

## Instrucciones

1. Determina el alcance de la revisión:
   - Si se pasaron argumentos (`$ARGUMENTS`), revisa solo esos archivos.
   - Si no hay argumentos, obtén la lista de archivos `.go` modificados con `git diff --name-only HEAD` y también los archivos sin seguimiento con `git ls-files --others --exclude-standard "*.go"`.
   - Si no hay cambios en git, revisa todos los archivos `.go` del proyecto.

2. Lee cada archivo `.go` en el alcance.

3. Evalúa cada archivo contra los siguientes patrones (el detalle completo está en `docs/security/patrones-seguros.md`):

   **P-001 — URLs de instancia Atlassian hardcodeadas** (MEDIUM)
   - Busca literales que contengan dominios corporativos, `.atlassian.net`, intranet o URLs de SaaS privado.
   - Excepción: strings en comentarios o textos de ayuda al usuario (ej: `"ej: https://empresa.atlassian.net"`).

   **P-002 — Env vars como bypass de seguridad** (HIGH)
   - Busca `os.Getenv(...)` cuyo valor se use para saltar autenticación, autorización o validación de integridad.
   - Las env vars para configuración (credenciales, URLs, log levels) son válidas.

   **P-003 — Identificadores de instancia Atlassian hardcodeados** (MEDIUM)
   - Busca project keys, space keys, board IDs o cualquier identificador organizacional de Jira/Confluence como literales en JQL, paths de API o lógica.
   - El valor debe venir de `config`, `ProjectConfig` o flag CLI — nunca quemado en código.

   **P-004 — VCS host/workspace hardcodeados** (MEDIUM)
   - Busca URLs de Bitbucket/GitHub/GitLab con organización específica como literales en `builder.go` u otros archivos.
   - Deben venir de `ProjectConfig.VCSHost` y `ProjectConfig.VCSOrg`.

   **P-005 — exec.Command sin LookPath** (MEDIUM)
   - Busca `exec.Command("git"`, `exec.Command("sh"` u otros ejecutables con nombre relativo.
   - Debe ir precedido de `exec.LookPath(...)` y el resultado usarse como primer argumento.

   **P-006 — Binarios descargados sin verificación de hash** (HIGH)
   - En código de descarga (`http.Get`, `http.Do`), verifica que haya `sha256.Sum256` y comparación de hash antes de `os.WriteFile` con permisos ejecutables.

   **P-007 — Falso positivo de scanner** (INFO)
   - Si encuentras strings como `"token"`, `"password"`, `"secret"` en textos de ayuda o nombres de flags, confirma que son falsos positivos (no son valores reales).
   - Reporta solo si el string parece un secreto real (34+ caracteres alfanuméricos, prefijos conocidos como `ghp_`, `ATATT3x`, `sk-`).

4. Presenta los resultados con este formato:

```
## Resultado /security-check

### ✓ Sin hallazgos  /  ⚠ N hallazgo(s)

| Patrón | Archivo | Línea | Descripción | Severidad |
|--------|---------|-------|-------------|-----------|
| P-003  | internal/atlassian/jira_qa.go | 35 | `project = APP` hardcodeado en JQL | MEDIUM |

### Hallazgos pre-existentes (no introducidos en este cambio)
Lista los hallazgos que ya existían antes del cambio actual (presentes en código no modificado).

### Conclusión
Una línea: si el código nuevo es seguro para commitear o si hay algo que corregir primero.
```

5. Si encuentras hallazgos HIGH, recomienda corregirlos antes de continuar. Para MEDIUM, menciona si es deuda técnica aceptable o requiere corrección inmediata según el contexto.
