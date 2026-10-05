---
id: <id del hallazgo>
modulo: <modulo del hallazgo, o el deducido de la ruta>
tipo: <hotspot | sin-test | marcador | archivo-grande | dependencia>
repo: <ruta del repo>
archivo: <ruta relativa del archivo principal>
puntos: <n>
fecha: <AAAA-MM-DD>
estado: propuesta
---

# <Título: verbo + qué + dónde, ≤ 100 caracteres>

**Rama:** `<rama con el formato de local.md, o <CLAVE>-XXXX-<slug>>`
**Estimación:** <n> pts (~<h> h)
**Repo:** <nombre del repo>
**Módulo:** <módulo del hallazgo; si el cambio toca otros, se indica en Riesgos>
**Origen:** `gtt backlog scan` · hallazgo `<id>` (<tipo>, severidad <severidad>) · <AAAA-MM-DD>

## Contexto
<Por qué esta tarea y por qué ahora, con datos:>
- `<archivo>` tuvo **<N> commits** en los últimos 90 días y tiene **<L> líneas**.
- Commits recientes que lo tocaron: <claves de Jira o hashes>.
- Los métodos que más se repiten en esos cambios: `<metodoA>`, `<metodoB>`.
- Tests actuales: <qué está cubierto y qué no / "ningún test menciona la clase">.

## Objetivo
<Una o dos frases: el resultado observable al terminar.>

## Alcance
1. **Análisis:** <qué revisar y documentar antes de cambiar nada>.
2. **Cambio:** <qué extraer / ajustar, con nombres concretos de clases y métodos>.
3. **Tests:** <qué tests unitarios agregar, qué casos cubren y cómo se aíslan (mocks)>.
4. **Verificación:** <correr la suite del módulo y comprobar que no cambia el comportamiento>.

## Fuera de alcance
- <Lo que queda para otra tarea, para que esta no crezca.>

## Criterios de aceptación
- [ ] <Criterio comprobable por un revisor.>
- [ ] <Criterio comprobable por un revisor.>
- [ ] Tests nuevos pasan y la suite del módulo sigue en verde.
- [ ] Sin cambios de comportamiento visibles para el usuario (o descritos explícitamente).
- [ ] Cumple los estándares del repo (<skills o documentos aplicables>).

## Archivos involucrados
| Archivo | Cambio |
|---|---|
| `<ruta>` | <modificar / nuevo> — <qué> |
| `<ruta test>` | **Nuevo** — <qué cubre> |

## Riesgos y mitigación
- **<Riesgo>:** <cómo se mitiga>.

## Cómo probar
```
<comandos exactos del repo>
```

## Antes de empezar
- Confirmar en Jira o con el equipo que nadie esté trabajando `<archivo>` en otra rama.
- No probar sobre datos reales: usar datos de prueba o mocks.
