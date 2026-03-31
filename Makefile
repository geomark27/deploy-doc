# ================================================================
#  deploy-doc - Makefile
# ================================================================

BINARY_NAME=deploy-doc
BUILD_DIR=./bin
MAIN=./main.go
MODULE=github.com/geomark27/deploy-doc
BRANCH := $(shell git branch --show-current 2>/dev/null || echo "main")
VERSION_PKG=github.com/geomark27/deploy-doc/internal/build

.DEFAULT_GOAL=help

# ================================================================
#  Colores
# ================================================================

CYAN=\033[0;36m
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m

# ----------------------------------------------------------------
#  Help
# ----------------------------------------------------------------

.PHONY: help
help:
	@echo ""
	@echo "$(CYAN)  deploy-doc - Generador de documentos de despliegue$(NC)"
	@echo ""
	@echo "  Uso: make <comando>"
	@echo ""
	@echo "$(YELLOW)  🔨 Build & Run:$(NC)"
	@echo "    build       Compila el binario en ./bin/deploy-doc"
	@echo "    build-all   Compila para Linux, Windows y Mac"
	@echo "    install     Compila e instala globalmente (~/.local/bin)"
	@echo "    run         Ejecuta el CLI sin compilar (dev mode)"
	@echo "    clean       Elimina los binarios compilados"
	@echo ""
	@echo "$(YELLOW)  🧪 Calidad de código:$(NC)"
	@echo "    fmt         Formatea el código fuente (go fmt)"
	@echo "    vet         Analiza el código en busca de errores (go vet)"
	@echo "    lint        Ejecuta fmt + vet juntos"
	@echo "    tidy        Limpia y actualiza dependencias (go mod tidy)"
	@echo ""
	@echo "$(YELLOW)  📦 Git ($(BRANCH)):$(NC)"
	@echo "    push m='msg'    Agrega, commitea y pushea a origin/$(BRANCH)"
	@echo "    pull            Hace pull desde origin/$(BRANCH)"
	@echo "    sync m='msg'    Pull + commit + push en un solo paso"
	@echo "    status          Muestra el estado del repositorio"
	@echo "    log             Muestra los últimos commits"
	@echo ""
	@echo "$(YELLOW)  🚀 Versioning & Release:$(NC)"
	@echo "    version         Ver el tag de versión actual"
	@echo "    release         Bump patch + compilar + push a GitHub"
	@echo "    release-minor   Bump minor version + push a GitHub"
	@echo "    release-major   Bump major version + push a GitHub"
	@echo ""
	@echo "$(YELLOW)  💡 Ejemplos:$(NC)"
	@echo "    make build"
	@echo "    make install"
	@echo "    make run ARGS='generate --issue APP-1999 --commit-backend 27cefd86'"
	@echo "    make push m='feat: agrego comando X'"
	@echo "    make sync m='fix: corrijo validación'"
	@echo ""

# ----------------------------------------------------------------
#  Build
# ----------------------------------------------------------------

.PHONY: build
build:
	@echo "$(YELLOW)Compilando...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -trimpath -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN)
	@echo "$(GREEN)✓ Binario generado en $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

.PHONY: build-all
build-all:
	@echo "$(YELLOW)Compilando para todos los sistemas operativos...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "-s -w -X $(VERSION_PKG).Version=$(VER)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN)
	@GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -X $(VERSION_PKG).Version=$(VER)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN)
	@GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "-s -w -X $(VERSION_PKG).Version=$(VER)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN)
	@echo "$(GREEN)✓ Binarios generados en $(BUILD_DIR)/$(NC)"

# ----------------------------------------------------------------
#  Install (disponible globalmente en la terminal)
# ----------------------------------------------------------------

.PHONY: install
install: build
	@mkdir -p ~/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Instalado en ~/.local/bin/$(BINARY_NAME)$(NC)"
	@echo ""
	@echo "  Si el comando no está disponible, agrega esto a tu ~/.zshrc o ~/.bashrc:"
	@echo "  export PATH=\$$PATH:~/.local/bin"

# ----------------------------------------------------------------
#  Run (dev mode, sin compilar)
# ----------------------------------------------------------------

.PHONY: run
run:
	@go run $(MAIN) $(ARGS)

# ----------------------------------------------------------------
#  Code quality
# ----------------------------------------------------------------

.PHONY: fmt
fmt:
	@echo "$(YELLOW)Formateando código...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Listo$(NC)"

.PHONY: vet
vet:
	@echo "$(YELLOW)Analizando código...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Sin errores$(NC)"

.PHONY: lint
lint: fmt vet

# ----------------------------------------------------------------
#  Dependencies
# ----------------------------------------------------------------

.PHONY: tidy
tidy:
	@echo "$(YELLOW)Actualizando dependencias...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ Listo$(NC)"

# ----------------------------------------------------------------
#  Clean
# ----------------------------------------------------------------

.PHONY: clean
clean:
	@echo "$(YELLOW)Limpiando...$(NC)"
	@rm -rf $(BUILD_DIR)
	@echo "$(GREEN)✓ Listo$(NC)"

# ================================================================
#  Git
# ================================================================

.PHONY: push
push:
	@if [ -z "$(m)" ]; then \
		echo "$(RED)❌ Error: Debes proporcionar un mensaje$(NC)"; \
		echo "   Uso: make push m='tu mensaje de commit'"; \
		exit 1; \
	fi
	@echo "$(YELLOW)📦 Agregando archivos...$(NC)"
	@git add .
	@echo "$(YELLOW)✏️  Commiteando: $(m)$(NC)"
	@git commit -m "$(m)"
	@echo "$(YELLOW)🚀 Pusheando a origin/$(BRANCH)...$(NC)"
	@git push origin $(BRANCH)
	@echo "$(GREEN)✓ Push completado!$(NC)"

.PHONY: pull
pull:
	@echo "$(YELLOW)⬇️  Pulling desde origin/$(BRANCH)...$(NC)"
	@git fetch origin
	@git pull origin $(BRANCH)
	@echo "$(GREEN)✓ Pull completado!$(NC)"

.PHONY: sync
sync:
	@if [ -z "$(m)" ]; then \
		echo "$(RED)❌ Error: Debes proporcionar un mensaje$(NC)"; \
		echo "   Uso: make sync m='tu mensaje de commit'"; \
		exit 1; \
	fi
	@echo "$(YELLOW)⬇️  Pulling cambios...$(NC)"
	@git pull origin $(BRANCH)
	@echo "$(YELLOW)📦 Agregando archivos...$(NC)"
	@git add .
	@echo "$(YELLOW)✏️  Commiteando: $(m)$(NC)"
	@git commit -m "$(m)"
	@echo "$(YELLOW)🚀 Pusheando a origin/$(BRANCH)...$(NC)"
	@git push origin $(BRANCH)
	@echo "$(GREEN)✓ Sincronización completada!$(NC)"

.PHONY: status
status:
	@echo "$(CYAN)📊 Estado de Git (rama: $(BRANCH)):$(NC)"
	@echo ""
	@git status

.PHONY: log
log:
	@echo "$(CYAN)📋 Últimos commits (rama: $(BRANCH)):$(NC)"
	@echo ""
	@git log --oneline -10

# ================================================================
#  Versioning & Release
# ================================================================

.PHONY: version
version:
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	echo "$(CYAN)Versión actual: $$CURRENT$(NC)"

.PHONY: release
release: lint
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT | cut -d. -f1 | tr -d 'v'); \
	MINOR=$$(echo $$CURRENT | cut -d. -f2); \
	PATCH=$$(echo $$CURRENT | cut -d. -f3); \
	NEW_PATCH=$$((PATCH + 1)); \
	NEW_TAG="v$$MAJOR.$$MINOR.$$NEW_PATCH"; \
	echo "$(CYAN)╔════════════════════════════════════════════════════════════╗$(NC)"; \
	echo "$(CYAN)║                  RELEASE - INICIANDO                      ║$(NC)"; \
	echo "$(CYAN)╚════════════════════════════════════════════════════════════╝$(NC)"; \
	echo ""; \
	echo "$(YELLOW)  Versión actual : $$CURRENT$(NC)"; \
	echo "$(GREEN)  Nueva versión  : $$NEW_TAG$(NC)"; \
	echo ""; \
	echo "$(YELLOW)[0/3]$(NC) Compilando binarios con version $$NEW_TAG..."; \
	$(MAKE) build-all VER=$$NEW_TAG; \
	echo ""; \
	echo "$(YELLOW)[1/3]$(NC) Commiteando y creando tag $$NEW_TAG..."; \
	git add -A; \
	git commit -m "release: $$NEW_TAG" 2>/dev/null || true; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
	echo "$(GREEN)✓ Tag creado$(NC)"; \
	echo ""; \
	echo "$(YELLOW)[2/3]$(NC) Subiendo a GitHub..."; \
	git push origin $(BRANCH) --tags; \
	echo "$(GREEN)✓ Push completado$(NC)"; \
	echo ""; \
	echo "$(YELLOW)[3/3]$(NC) Publicando binarios en GitHub Releases..."; \
	gh release create $$NEW_TAG \
		$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 \
		$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		--title "$$NEW_TAG" \
		--notes "Release $$NEW_TAG"; \
	echo ""; \
	echo "$(GREEN)✓ Release $$NEW_TAG completado$(NC)"; \
	echo "$(CYAN)  https://github.com/geomark27/deploy-doc/releases/tag/$$NEW_TAG$(NC)"

.PHONY: release-minor
release-minor: lint
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT | cut -d. -f1 | tr -d 'v'); \
	MINOR=$$(echo $$CURRENT | cut -d. -f2); \
	NEW_MINOR=$$((MINOR + 1)); \
	NEW_TAG="v$$MAJOR.$$NEW_MINOR.0"; \
	$(MAKE) build-all VER=$$NEW_TAG; \
	git add -A; \
	git commit -m "release: $$NEW_TAG" 2>/dev/null || true; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
	git push origin $(BRANCH) --tags; \
	gh release create $$NEW_TAG \
		$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 \
		$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		--title "$$NEW_TAG" \
		--notes "Release $$NEW_TAG"; \
	echo "$(GREEN)✓ Release minor $$NEW_TAG completado$(NC)"

.PHONY: release-major
release-major: lint
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT | cut -d. -f1 | tr -d 'v'); \
	NEW_MAJOR=$$((MAJOR + 1)); \
	NEW_TAG="v$$NEW_MAJOR.0.0"; \
	$(MAKE) build-all VER=$$NEW_TAG; \
	git add -A; \
	git commit -m "release: $$NEW_TAG" 2>/dev/null || true; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG"; \
	git push origin $(BRANCH) --tags; \
	gh release create $$NEW_TAG \
		$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 \
		$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		--title "$$NEW_TAG" \
		--notes "Release $$NEW_TAG"; \
	echo "$(GREEN)✓ Release major $$NEW_TAG completado$(NC)"
