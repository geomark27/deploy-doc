#!/usr/bin/env bash
# version-check — estado de versiones de gtt antes de cerrar una tarea o liberar.
#
# Solo lee: no crea tags, no commitea, no publica. Reproduce el cálculo de
# versión del Makefile (git describe --tags --abbrev=0 desde HEAD) para que lo
# que se reporta sea exactamente lo que haría `make release*`.
#
# Uso:   bash .claude/skills/version-check/check.sh
# Salida: líneas ✓ / ⚠ / ✗ y un dictamen. Exit 1 solo si hay bloqueantes ✗.

set -u
cd "$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "✗ No es un repositorio git"; exit 1; }

ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m⚠\033[0m %s\n' "$*"; WARNINGS=$((WARNINGS + 1)); }
fail() { printf '  \033[31m✗\033[0m %s\n' "$*"; BLOCKERS=$((BLOCKERS + 1)); }
info() { printf '  · %s\n' "$*"; }
section() { printf '\n\033[1m%s\033[0m\n' "$*"; }

WARNINGS=0
BLOCKERS=0
RELEASE_BRANCH=main
BITACORA=docs/bitacora

# bump <tag> <patch|minor|major> — same arithmetic as the Makefile targets.
bump() {
	local v=${1#v} major minor patch
	IFS=. read -r major minor patch <<<"$v"
	case $2 in
		patch) echo "v$major.$minor.$((patch + 1))" ;;
		minor) echo "v$major.$((minor + 1)).0" ;;
		major) echo "v$((major + 1)).0.0" ;;
	esac
}

# documented <tag> — is the tag mentioned in any bitácora entry?
documented() { grep -rqF -- "$1" "$BITACORA" 2>/dev/null; }

# ── 1. Remoto ────────────────────────────────────────────────────────────────
section "1. Remoto"
if git fetch --quiet --tags origin 2>/dev/null; then
	ok "Tags y ramas actualizados desde origin"
else
	warn "No se pudo hacer fetch de origin: el resultado usa los tags locales"
fi

# ── 2. Rama y working tree ───────────────────────────────────────────────────
section "2. Rama y working tree"
BRANCH=$(git branch --show-current)
if [ "$BRANCH" = "$RELEASE_BRANCH" ]; then
	ok "Rama actual: $BRANCH"
	if [ "$(git rev-parse HEAD)" != "$(git rev-parse "origin/$RELEASE_BRANCH" 2>/dev/null)" ]; then
		warn "$BRANCH local no coincide con origin/$BRANCH: haz git pull antes de liberar"
	fi
else
	info "Rama actual: $BRANCH"
	warn "make release* NO debe correrse aquí: commitea, tagea y hace push sobre la rama actual. Libera desde $RELEASE_BRANCH"
fi

DIRTY=$(git status --porcelain)
if [ -z "$DIRTY" ]; then
	ok "Working tree limpio"
else
	warn "Cambios sin commit: make release hace 'git add -A' y los metería en el commit de release:"
	git status --short | sed 's/^/      /'
fi

# ── 3. Tags ──────────────────────────────────────────────────────────────────
section "3. Tags"
LATEST=$(git tag -l 'v*' --sort=-v:refname | head -1)
DESCRIBED=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
if [ -z "$LATEST" ]; then
	warn "No hay tags v*: el primer release saldrá desde v0.0.0"
	LATEST=v0.0.0
fi
info "Último tag del repo:              $LATEST"
info "Base que usará make release*:     $DESCRIBED (git describe desde HEAD)"
if [ "$DESCRIBED" != "$LATEST" ]; then
	fail "HEAD no contiene $LATEST: make release partiría de $DESCRIBED y podría repetir una versión ya publicada"
fi

# Tags outside the release branch or sharing a commit with another tag.
for tag in $(git tag -l 'v*' --sort=-v:refname | head -10); do
	commit=$(git rev-list -n1 "$tag")
	if ! git merge-base --is-ancestor "$commit" "origin/$RELEASE_BRANCH" 2>/dev/null; then
		if documented "$tag"; then
			info "$tag no está en $RELEASE_BRANCH (documentado en la bitácora)"
		else
			warn "$tag apunta a un commit que no está en $RELEASE_BRANCH y la bitácora no lo explica"
		fi
	fi
	twins=$(git tag --points-at "$commit" | grep -v -x -F "$tag" | tr '\n' ' ')
	if [ -n "$twins" ] && ! documented "$tag"; then
		warn "$tag comparte commit con: $twins— sin nota en la bitácora"
	fi
done

# ── 4. GitHub Releases ───────────────────────────────────────────────────────
section "4. GitHub Releases"
if command -v gh >/dev/null 2>&1; then
	GH_LATEST=$(gh release list --limit 20 --json tagName,isLatest -q '.[] | select(.isLatest) | .tagName' 2>/dev/null)
	if [ -z "$GH_LATEST" ]; then
		warn "No se pudo consultar GitHub Releases (¿gh auth login?)"
	elif [ "$GH_LATEST" = "$LATEST" ]; then
		ok "Release 'Latest' en GitHub: $GH_LATEST (coincide con el último tag)"
	else
		warn "Release 'Latest' en GitHub es $GH_LATEST pero el último tag es $LATEST"
	fi
	for tag in $(git tag -l 'v*' --sort=-v:refname | head -5); do
		gh release view "$tag" >/dev/null 2>&1 || warn "$tag existe como tag pero no tiene release publicado (gtt update no lo ve)"
	done
else
	info "gh no está instalado: se omite la comparación con GitHub Releases"
fi

# ── 5. Cambios pendientes y próxima versión ──────────────────────────────────
section "5. Cambios pendientes de liberar"
PENDING=$(git log --oneline --no-merges "$DESCRIBED..HEAD" -- . ":(exclude)$BITACORA" 2>/dev/null)
if [ -z "$PENDING" ] && [ -z "$DIRTY" ]; then
	ok "Nada pendiente desde $DESCRIBED"
else
	[ -n "$PENDING" ] && { info "Commits desde $DESCRIBED:"; echo "$PENDING" | sed 's/^/      /'; }
	[ -n "$DIRTY" ] && info "Más los cambios sin commit de la sección 2"
	info "Próxima versión según el target:"
	info "  make release        → $(bump "$DESCRIBED" patch)   (fix, docs, ayuda, refactor sin cambio visible)"
	info "  make release-minor  → $(bump "$DESCRIBED" minor)   (comando o flag nuevo, comportamiento nuevo)"
	info "  make release-major  → $(bump "$DESCRIBED" major)   (rompe flags, config o el formato del JSON)"
fi

# ── 6. Bitácora ──────────────────────────────────────────────────────────────
section "6. Bitácora"
if [ -n "$PENDING$DIRTY" ]; then
	found=""
	for kind in patch minor major; do
		next=$(bump "$DESCRIBED" "$kind")
		[ -f "$BITACORA/$next.md" ] && found="$found $next"
	done
	if [ -z "$found" ]; then
		fail "Hay cambios pendientes y no existe $BITACORA/<próxima versión>.md ($(bump "$DESCRIBED" patch), $(bump "$DESCRIBED" minor) o $(bump "$DESCRIBED" major))"
	else
		for next in $found; do
			ok "Existe $BITACORA/$next.md"
			prev=$(grep -m1 -oE '\*\*Versión anterior:\*\* *v[0-9.]+' "$BITACORA/$next.md" | grep -oE 'v[0-9.]+$')
			if [ "$prev" = "$DESCRIBED" ]; then
				ok "$next.md declara 'Versión anterior: $prev'"
			else
				fail "$next.md declara 'Versión anterior: ${prev:-(falta)}' pero la base real es $DESCRIBED"
			fi
			if grep -qF "($next.md)" "$BITACORA/README.md" 2>/dev/null || grep -qF "(./$next.md)" "$BITACORA/README.md" 2>/dev/null; then
				ok "$next aparece en el índice $BITACORA/README.md"
			else
				warn "$next no aparece en el índice $BITACORA/README.md"
			fi
		done
		[ "$(echo $found | wc -w)" -gt 1 ] && warn "Hay más de una entrada candidata ($found): deja solo la del tipo de release que vas a hacer"
	fi
fi

# Entries of versions already published must not receive new content. The list
# is built first so warn() runs in this shell and its count is not lost.
TOUCHED=$({ git diff --name-only "$DESCRIBED" -- "$BITACORA" 2>/dev/null; git status --porcelain -- "$BITACORA" | awk '{print $2}'; } | sort -u)
for f in $TOUCHED; do
	ver=$(basename "$f" .md)
	if git rev-parse -q --verify "refs/tags/$ver" >/dev/null; then
		warn "Se modificó $f, de una versión ya publicada: solo notas aclaratorias; el contenido nuevo va en la próxima entrada"
	fi
done

# ── 7. gtt instalado ─────────────────────────────────────────────────────────
section "7. gtt instalado en este equipo"
for bin in gtt "${LOCALAPPDATA:-}/Programs/gtt/gtt.exe" /mnt/c/Users/*/AppData/Local/Programs/gtt/gtt.exe; do
	if command -v "$bin" >/dev/null 2>&1 || [ -x "$bin" ]; then
		installed=$("$bin" version 2>/dev/null </dev/null | awk '{print $2}')
		[ -z "$installed" ] && continue
		if [ "$installed" = "$LATEST" ]; then
			ok "$bin → $installed"
		else
			info "$bin → $installed (último: $LATEST; se actualiza con: gtt update)"
		fi
	fi
done

# ── Dictamen ─────────────────────────────────────────────────────────────────
section "Dictamen"
if [ "$BLOCKERS" -gt 0 ]; then
	printf '  \033[31m✗ NO listo para liberar\033[0m: %d bloqueante(s), %d advertencia(s)\n' "$BLOCKERS" "$WARNINGS"
	exit 1
fi
if [ "$WARNINGS" -gt 0 ]; then
	printf '  \033[33m⚠ Revisar\033[0m: %d advertencia(s), sin bloqueantes\n' "$WARNINGS"
else
	printf '  \033[32m✓ Versiones en orden\033[0m\n'
fi
exit 0
