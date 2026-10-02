---
name: version-check
description: Verifica el estado de versiones de gtt (tags, GitHub Releases, rama, working tree, próxima versión y bitácora) con un script de solo lectura. Úsala SIEMPRE al terminar cualquier tarea que modifique código, docs o el Makefile de este repo, antes de dar el trabajo por cerrado, y antes de cualquier `make release*`. También cuando el usuario pregunte "qué versión sigue", "puedo liberar", "está bien la bitácora" o mencione tags/releases.
---

# version-check

Comprueba que las versiones de `gtt` estén en orden y que el próximo release
salga bien. El script reproduce el cálculo del `Makefile`, así que la versión
que reporta es exactamente la que crearía `make release*`.

## Por qué existe

`make release`, `release-minor` y `release-major` hacen `git add -A`, commit,
tag y push **sobre la rama actual**. El 2026-10-01 se corrió desde
`feat/backlog-scan`: el código de la feature quedó en un commit llamado
`release: v1.3.1` y salieron tres releases (v1.3.1, v1.4.0, v1.4.1) para un mismo
cambio. Además, una entrada de bitácora de una versión ya publicada recibió
contenido de la siguiente. Esta skill existe para detectar esas situaciones
antes, no después.

## Cuándo usarla

1. **Al terminar cualquier tarea** que cambie archivos de este repo, como último
   paso antes de reportar al usuario. No esperes a que lo pida.
2. **Antes de `make release*`**, y dentro de `/pre-release` (paso 5).
3. Cuando el usuario pregunte por versiones, tags, releases o la bitácora.

## Pasos

1. Ejecuta el script desde la raíz del repo (en WSL/Linux; en Windows, con Git
   Bash o `wsl -e bash -lc "..."`):

   ```bash
   bash .claude/skills/version-check/check.sh
   ```

   Es de solo lectura: hace `git fetch --tags`, lee tags, ramas, la bitácora y,
   si `gh` está disponible, GitHub Releases. Nunca crea tags ni commitea.

2. Interpreta cada línea:

   | Marca | Significado | Qué hacer |
   |---|---|---|
   | ✓ | En orden | Nada |
   | ⚠ | Advertencia | Reportarla al usuario con la acción concreta |
   | ✗ | Bloqueante para liberar | Corregirla si es parte de la tarea (ej. crear la bitácora) o reportarla como pendiente |
   | · | Información | Usarla en el resumen (próxima versión, base del release) |

3. **Decide el tipo de release** a partir de los cambios pendientes que lista la
   sección 5, con este criterio:

   | Tipo | Cuándo | Target |
   |---|---|---|
   | patch | Fix, docs, ayuda, refactor sin cambio visible para el usuario | `make release` |
   | minor | Comando o flag nuevo, comportamiento nuevo compatible | `make release-minor` |
   | major | Rompe flags, nombres de comando, claves de config o el formato del JSON | `make release-major` |

4. **Bitácora** (`docs/bitacora/`):
   - Debe existir `<próxima versión>.md` **del tipo elegido**, con
     `**Versión anterior:**` igual a la base que reporta la sección 3.
   - Si el script detecta varias entradas candidatas (por ejemplo `v1.4.2.md` y
     `v1.5.0.md`), deja solo la del tipo que se va a liberar.
   - Debe tener su fila en el índice `docs/bitacora/README.md`.
   - **Nunca agregues contenido nuevo a la entrada de una versión ya publicada.**
     Solo se permiten notas aclaratorias (por ejemplo, explicar tags duplicados).
     Lo nuevo va en la entrada de la próxima versión.
   - Formato: Solicitud → Motivación → Diseño técnico → Archivos → Cómo verificar
     → Notas de compatibilidad (ver la última entrada como referencia).

5. **Cierra con un resumen corto** para el usuario:
   - Versión publicada actual (último tag y "Latest" en GitHub).
   - Próxima versión y target recomendado (`make release` / `-minor` / `-major`).
   - Bloqueantes y advertencias, cada una con su acción.
   - El recordatorio del orden correcto si hay algo pendiente de liberar:
     commit → PR → merge a `main` → `git checkout main && git pull` →
     `make release*`.

## Lo que esta skill no hace

- No corre `make release*`, no crea tags, no hace commit ni push: eso lo decide
  y ejecuta el usuario.
- No borra ni mueve tags ya publicados. Un tag de más se documenta en la
  bitácora; borrarlo rompe a quien ya lo descargó.
