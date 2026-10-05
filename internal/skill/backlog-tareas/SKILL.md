---
name: backlog-tareas
description: Convierte los hallazgos de `gtt backlog scan` en tareas listas para Jira, con el código real verificado, alcance acotado a las horas pedidas y texto detallado (rama, contexto, objetivo, alcance, criterios de aceptación, archivos, riesgos, cómo probar). Úsala cuando el usuario pida "una tarea de N horas", "qué hago hoy", "necesito completar las 8 horas", "dame tareas del backlog", "propón una tarea de deuda técnica" o mencione el backlog técnico. Solo propone y redacta: no modifica código, no crea ramas ni tickets.
---

# backlog-tareas

Toma los hallazgos de `gtt backlog scan` (archivos que cambian demasiado, clases
sin tests, marcadores pendientes, archivos enormes) y los convierte en **tareas
concretas, verificadas en el código y redactadas para pegar en Jira**.

Un hallazgo dice *dónde* hay deuda; la tarea tiene que decir *qué hacer, qué
entregar y cómo se comprueba*, en un tamaño que quepa en las horas pedidas.

> Esta skill la instala y actualiza `gtt` (`gtt skill status`). No la edites:
> los cambios se detectan y no se sobrescriben, pero quedas fuera de las
> mejoras. Lo propio de tu equipo va en [local.md](local.md).

## Convenciones del equipo: local.md

**Antes de empezar, lee [local.md](local.md)** en la carpeta de esta skill, si
existe. Ahí el usuario define lo que depende de su equipo y de sus repos:
equivalencia de puntos a horas, formato de rama, skills o documentos de
estándares por repo, comandos de prueba y ejemplos de módulos. **Lo que diga
`local.md` tiene prioridad sobre los valores por defecto de esta skill.** Si no
existe o no define algo, usa el valor por defecto indicado aquí.

## Reglas

- **Solo lectura.** No modifiques código, no crees ramas, no hagas commit, no
  crees tickets en Jira. El resultado es texto.
- **Verifica antes de proponer.** Cada tarea sale de leer el código real, no
  solo del JSON. Si el hallazgo no se sostiene, descártalo y di por qué.
- **Acota al tiempo pedido.** Equivalencia por defecto: **3 pts ≈ 4 h** (o la
  de `local.md`). Si pide otra cantidad de horas, prorratea (8 h ≈ dos tareas
  de 3 pts, o una de 5 pts bien justificada) y dilo explícitamente. Una tarea
  que no cabe se parte en incrementos; nunca propongas "refactorizar el archivo
  completo".
- **Respeta los estándares del repo.** Antes de redactar, lee el `AGENTS.md` /
  `CLAUDE.md` del repo y las skills o documentos de estándares que referencian
  (y los que `local.md` asocie a ese repo). El alcance y los criterios de
  aceptación deben cumplirlos (por ejemplo: cómo se escriben los tests, qué
  capas existen y en qué orden se llaman).
- **Datos.** Usa datos de prueba en los pasos de verificación; nunca pegues datos
  reales de clientes ni de operaciones en el texto de la tarea.

## Pasos

### 1. Entender el pedido
Extrae del mensaje del usuario, sin preguntar si se puede inferir:
- **Horas objetivo.** Por defecto 4 h (una tarea de 3 pts).
- **Repo.** `backend`, `frontend` o ambos. Por defecto, ambos.
- **Módulo.** Resuélvelo en este orden y **nunca elijas uno por tu cuenta**:
  1. El que el usuario menciona ("una tarea de Facturación").
  2. Si no menciona ninguno, mira los reportes de la carpeta (paso 2): si el más
     reciente es de un módulo (`<proyecto>-<repo>-<modulo>.json`), el usuario
     acaba de correr `gtt backlog scan -m <modulo> -o`. Usa ese módulo y dilo
     ("uso Facturación, el último scan que corriste").
  3. Si tampoco, pregunta en una línea, listando los módulos de `modulos[]` con
     su cantidad de hallazgos, o si prefiere "cualquier módulo".
- **Preferencias**, si las da: tipo de trabajo (tests, refactor, limpieza) o un
  archivo concreto.

### 2. Cargar los hallazgos
Los reportes viven en `%USERPROFILE%\.config\gtt\backlog\` (en Linux/macOS,
`~/.config/gtt/backlog/`), con dos nombres posibles:

| Archivo | Lo crea | Contiene |
|---|---|---|
| `<proyecto>-<repo>.json` | `gtt backlog scan -o` | Todos los módulos |
| `<proyecto>-<repo>-<modulo>.json` | `gtt backlog scan -m <modulo> -o` | Solo ese módulo (el nombre va en minúsculas y sin guiones) |

**Qué archivo usar:**
- **Con módulo:** el `-<modulo>.json` de cada repo. Si no existe, o tiene más de
  24 h (campo `generado`), regenera **solo ese módulo**:
  ```
  gtt backlog scan -m <modulo> -o
  ```
  Si no existe pero sí el reporte completo y está fresco, puedes filtrarlo por el
  campo `modulo` sin regenerar.
- **Sin módulo ("cualquiera"):** `<proyecto>-<repo>.json`. Si no existe o tiene
  más de 24 h: `gtt backlog scan -o`.
- **No regeneres lo que está fresco** y nunca corras un scan completo cuando hay
  módulo: crea archivos que el usuario no pidió.

Cada reporte es un objeto con
`version`, `repo`, `generado`, `desde`, `total`, `hallazgos[]` y, si el proyecto
configura `modules`, `modulos[]` (resumen por módulo); cada hallazgo tiene `id`,
`tipo`, `severidad`, `puntos`, `titulo`, `modulo`, `archivos`, `evidencia`,
`metrica`. Las rutas de `archivos` usan `/` y son relativas a `repo`.

Si los hallazgos no traen `modulo`, el proyecto no tiene `modules` en su
`config.yaml`: avísalo (ver `gtt help backlog`) y deduce el módulo de la ruta
(por ejemplo `app/Http/<Modulo>/`, `src/app/<modulo>/`).

Si el JSON trae `omitidos` con `sin-test`, avisa que activar `class_globs` en el
`config.yaml` daría hallazgos de tests (ver `gtt help backlog`).

### 3. Evitar repetir
Las tareas ya propuestas se guardan en
`%USERPROFILE%\.config\gtt\backlog\tareas\` (en Linux/macOS,
`~/.config/gtt/backlog/tareas/`). Lee el frontmatter de esos `.md` y
**no vuelvas a proponer un hallazgo con el mismo `id`** salvo que el usuario lo
pida. Un mismo archivo sí puede dar otra tarea si es **otro incremento**
(otra responsabilidad, otro flujo): dilo en el contexto ("segunda tarea sobre
este archivo; la anterior cubrió X").

### 4. Elegir candidatos
Ordena por este criterio y toma los primeros que sumen las horas pedidas:
1. `sin-test` con severidad alta (cambia mucho y no tiene tests: el mayor riesgo).
2. `hotspot` alta (cambia mucho y es grande).
3. `hotspot` media.
4. `marcador` con FIXME/HACK (defecto conocido).
5. `archivo-grande` y `marcador` TODO.

**El módulo es un filtro estricto:** solo entran hallazgos cuyo `modulo` sea el
pedido (sin distinguir mayúsculas ni guiones). Un hallazgo de otro módulo nunca
se propone en su lugar, aunque tenga más prioridad. Si el módulo no tiene
candidatos suficientes para las horas pedidas, dilo y pregunta si completar con
otro módulo o con una tarea más chica.

Aplica después las preferencias del usuario. Si una tarea toca otros módulos
(dependencias, servicios compartidos), dilo en "Riesgos y mitigación": el módulo
de la tarea es el del archivo principal.

### 5. Investigar cada candidato en el código
Abre el archivo en `repo` y responde, con evidencia:
- **Qué cambia tanto.** Mira los commits del período sobre el archivo
  (`git log --since="90 days ago" --format="%h %s" -- <archivo>` y
  `git show <hash> -- <archivo>`) e identifica los **métodos que más se
  repiten** en esos cambios. Ese es el flujo a proteger o extraer.
- **Qué tests hay.** Busca la clase en la carpeta de tests y lista qué métodos
  están cubiertos y cuáles no.
- **De qué depende.** Dependencias del constructor, modelos, servicios
  externos, transacciones. Define si el test puede ser unitario con mocks.
- **Trabajo en paralelo.** `git log --all --since="14 days ago" --format="%h %d %s" -- <archivo>`:
  si otra rama lo está tocando, adviértelo como riesgo de conflicto.

Con eso define **un incremento concreto**, por ejemplo:
- *"Extraer el cálculo de descuentos por volumen (`calcularDescuento`,
  `aplicarTopeDescuento`) a `CalculadoraDescuentos` y cubrir sus 4 casos con
  tests unitarios"*, no *"extraer una responsabilidad"*.
- *"Cubrir con tests `registrarPedido` en los casos pedido nuevo, pedido
  duplicado y reasignación de bodega"*, no *"agregar tests"*.

Si el candidato no se sostiene (código generado, ya en refactor en otra rama,
cambio de bajo valor), descártalo, dilo en una línea y pasa al siguiente.

### 6. Redactar
Usa **exactamente** la plantilla de [plantilla.md](plantilla.md), sección por
sección. Reglas de redacción:
- **Título (Summary de Jira):** verbo en infinitivo + qué + dónde, ≤ 100
  caracteres. Ej.: "Extraer cálculo de descuentos de PedidoService".
- **Rama:** el formato de `local.md`. Si no define uno:
  `<CLAVE>-XXXX-<slug-en-minúsculas-con-guiones>`, con `XXXX` literal para que
  el usuario lo reemplace por el número del ticket y `<CLAVE>` la clave del
  proyecto en Jira si se conoce (si no, déjala literal).
- **Contexto:** los datos que justifican la tarea (commits en el período,
  líneas, tests existentes, commits de ejemplo). Es lo que respalda las horas.
- **Alcance:** pasos numerados y verificables. Siempre incluye el de tests.
- **Fuera de alcance:** lo que se deja explícitamente para otra tarea. Es lo
  que mantiene la tarea en su tamaño.
- **Criterios de aceptación:** casillas `- [ ]` comprobables por un revisor
  (no "código más limpio", sí "`PedidoService` delega en
  `CalculadoraDescuentos` y ya no contiene `calcularDescuento`").
- **Cómo probar:** comandos exactos del repo, tomados de `local.md`, del
  `AGENTS.md` / `CLAUDE.md` del repo o de su configuración de tests.

El texto va en Markdown: el editor de Jira Cloud lo convierte al pegarlo
(títulos, listas, casillas y bloques de código).

### 7. Guardar y entregar
1. Guarda cada tarea en
   `%USERPROFILE%\.config\gtt\backlog\tareas\<AAAA-MM-DD>-<id>.md` (en
   Linux/macOS, `~/.config/gtt/backlog/tareas/`), con el frontmatter de la
   plantilla (`estado: propuesta`). Es local: no la guardes en carpetas
   sincronizadas (OneDrive) ni dentro del repo.
2. Muestra en el chat, por tarea: el título, la estimación y el bloque completo
   listo para copiar. Indica además el comando para copiarla al portapapeles
   sin el frontmatter (que es para la skill, no para Jira). En Windows:
   ```powershell
   (Get-Content "<ruta del .md>" -Raw -Encoding UTF8) -replace '(?s)^---.*?---\s*', '' | Set-Clipboard
   ```
   En Linux/macOS, que copie desde el título (`# ...`) hasta el final.
3. Cierra con una línea de resumen: **módulo usado y de dónde salió** (pedido,
   último scan o elegido por el usuario), **reporte(s) leídos** y si se
   regeneraron, horas cubiertas, tareas propuestas y candidatos descartados con
   el motivo.
