# Bitácora de versiones — gtt (deploy-doc)

Historial de versiones con contexto de solicitud, decisiones técnicas e implementación.
Cada entrada sigue el formato: **Solicitud → Motivación → Diseño técnico → Archivos → Verificación**.

---

## Índice de versiones

| Versión | Fecha | Tipo | Descripción |
|---|---|---|---|
| [v1.0.x LTS](./v1.0.x-lts.md) | 2025 | **LTS** | Primera versión en producción — baseline del proyecto |
| [v1.1.0](./v1.1.0.md) | 2026-04-03 | Minor | BitbucketOrg dinámico, multi-commit, `--dry-run`, warning init, búsqueda x10 |
| [v1.1.1](./v1.1.1.md) | 2026-04-10 | Patch | Renombre a `gtt`, flags cortos, UI con colores y pasos, fix downgrade en update |
| [v1.1.5](./v1.1.5.md) | 2026-05-04 | **Security Patch** | Remediación DevSecOps: backdoor GTT_DEV, URLs/IDs hardcodeados, verificación de integridad en update |
| [v1.2.0](./v1.2.0.md) | 2026-05-04 | Minor | Eliminación total del legado deploy-doc: binarios, paths, instalador y migración automática de config |

---

## Formato de cada entrada

Secciones obligatorias:

1. **Solicitud** — qué pidió el usuario/equipo en lenguaje natural
2. **Motivación** — por qué era necesario el cambio
3. **Diseño técnico** — decisiones tomadas y sus razones
4. **Archivos modificados** — lista de archivos con descripción del cambio
5. **Cómo verificar** — comandos concretos para validar cada cambio
