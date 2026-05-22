---
description: Checklist completa pre-release de gtt. Ejecuta lint, build-all, revisión de seguridad, verifica checksums (P-006) y estado del repo. Termina con dictamen GO o NO-GO.
---

# /pre-release

Ejecuta la checklist completa antes de un release de `gtt`. Verifica calidad, seguridad e integridad del build. Al final da un dictamen claro: **GO** o **NO-GO**.

## Instrucciones

Ejecuta estos pasos en orden. Si un paso falla, detente y reporta el problema antes de continuar.

### Paso 1 — Lint y formato
```bash
make lint
```
- Si hay errores de `go fmt` o `go vet`, muéstralos y detente.

### Paso 2 — Build multiplataforma
```bash
make build-all
```
- Confirma que los binarios para Linux, Windows y Mac se generaron en `bin/`.

### Paso 3 — Revisión de seguridad completa
Ejecuta `/security-check` sobre todos los archivos `.go` del proyecto (sin argumentos, modo completo).
- Cualquier hallazgo **HIGH** es un NO-GO inmediato.
- Los hallazgos **MEDIUM** pre-existentes documentados en `docs/security/patrones-seguros.md` son aceptables.
- Cualquier hallazgo **MEDIUM nuevo** (no documentado) requiere decisión explícita del usuario.

### Paso 4 — Verificar generación de checksums (P-006)
Lee el `Makefile` y verifica que el target `release` genere un archivo `checksums.txt` con hashes SHA-256 de los binarios publicados.
- Si no existe, reportarlo como **bloqueante** (viola P-006).
- Muestra el comando exacto que debería agregarse al Makefile si falta.

### Paso 5 — Verificar versión
- Lee `internal/build/version.go` y muestra la versión actual.
- Confirma que el tag de git más reciente coincide: `git describe --tags --abbrev=0`.
- Si hay diferencia, advierte al usuario.

### Paso 6 — Estado del repositorio
```bash
git status
git log --oneline -5
```
- Si hay cambios sin commitear, reportar como advertencia (no bloqueante, pero el usuario debe confirmarlo).

---

## Dictamen final

Presenta una tabla resumen:

| Paso | Estado | Detalle |
|------|--------|---------|
| Lint | ✓ / ✗ | ... |
| Build-all | ✓ / ✗ | ... |
| Seguridad | ✓ / ⚠ / ✗ | N hallazgos |
| Checksums | ✓ / ✗ | ... |
| Versión | ✓ / ⚠ | ... |
| Git status | ✓ / ⚠ | ... |

**Dictamen: GO ✓** — listo para ejecutar `make release`
— o —
**Dictamen: NO-GO ✗** — descripción de lo que hay que resolver primero
