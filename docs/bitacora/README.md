# Bitácora de versiones — deploy-doc

Historial de versiones con contexto de solicitud, decisiones técnicas e implementación.
Cada entrada sigue el formato: **Solicitud → Motivación → Diseño técnico → Archivos → Verificación**.

---

## Índice de versiones

| Versión | Fecha | Tipo | Descripción |
|---|---|---|---|
| [v1.0.x LTS](./v1.0.x-lts.md) | 2025 | **LTS** | Primera versión en producción — baseline del proyecto |
| [v1.1.0](./v1.1.0.md) | 2026-04-03 | Minor | BitbucketOrg dinámico, multi-commit, `--dry-run`, warning init, búsqueda x10 |

---

## Formato de cada entrada

Secciones obligatorias:

1. **Solicitud** — qué pidió el usuario/equipo en lenguaje natural
2. **Motivación** — por qué era necesario el cambio
3. **Diseño técnico** — decisiones tomadas y sus razones
4. **Archivos modificados** — lista de archivos con descripción del cambio
5. **Cómo verificar** — comandos concretos para validar cada cambio
