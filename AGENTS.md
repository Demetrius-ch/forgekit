# AGENTS.md — ForgeKit

## Aperçu du projet
ForgeKit est un outil CLI en Go qui génère des APIs REST prêtes pour la production avec une architecture hexagonale. Il scaffolde des projets avec PostgreSQL, Docker, migrations, tests, et fournit les commandes `doctor`, `check`, `analyze`, `inspect`, `config`, `add`, `remove`, `version` pour la validation et l'extension.

## Commandes de build et test

```bash
# Build du CLI
go build -o forge ./cmd/forge

# Lancer tous les tests (ordre CI : vet -> test -> build)
go vet ./...
go test ./...
go build ./...

# Lancer les tests d'un package spécifique
go test ./internal/generator/...
go test ./internal/feature/...
go test ./internal/cli/...
go test ./internal/output/...
go test ./internal/report/...
go test ./internal/rules/...
go test ./internal/forge/...
go test ./internal/dbinspect/...
go test ./internal/ports/...

# Lancer les tests e2e (nécessite le build tag)
go test -tags=e2e ./internal/e2e/...

# Test d'intégration (exécuté en CI)
./forge init ci-api --non-interactive --module github.com/Demetrius-ch/ci-api
cd ci-api && go test ./...
```

## Cibles Makefile
```bash
make build          # Build le binaire forge (développement)
make test           # Lance tous les tests
make vet            # Lance go vet
make lint           # Lance gofmt -w .
make build-linux    # Build binaire Linux statique (CGO_ENABLED=0)
make package        # Build artefacts de release avec GoReleaser (snapshot)
make clean          # Supprime les artefacts de build
make ci             # Pipeline CI complet (lint -> vet -> test -> build)
```

## Architecture clé

- **cmd/forge/main.go** — Point d'entrée, délègue à `internal/cli`
- **internal/cli/commands.go** — Toutes les commandes CLI (init, add, remove, doctor, check, analyze, inspect, config, version)
- **internal/cli/root.go** — Configuration commande racine, flags globaux, branding
- **internal/generator/** — Logique de génération de projet (templates dans `internal/template/`)
- **internal/template/api/** — Fichiers Go text/template pour la structure du projet généré
- **internal/feature/** — Registre de features pour `forge add`/`forge remove` (interface, detector, installer, registry)
- **internal/feature/auth/** — Implémentation feature authentification JWT
- **internal/feature/cors/** — Implémentation feature middleware CORS
- **internal/feature/logging/** — Implémentation feature logging
- **internal/feature/swagger/** — Implémentation feature Swagger/OpenAPI
- **internal/rules/** — Règles architecturales pour check/analyze (sécurité, architecture, qualité, environnement, docker, config)
- **internal/analyzer/project.go** — Logique d'analyse de projet
- **internal/app/** — Métadonnées de l'app (nom, version, slogan)
- **internal/config/** — Config utilisateur (~/.forgekit/config.yaml) et config projet (forge.yaml)
- **internal/output/** — Utilitaires console (spinner, sortie colorée, JSON)
- **internal/engine/** — Moteur d'exécution de templates avec variables
- **internal/report/** — Scoring et reporting pour analyze
- **internal/errs/** — Types d'erreurs
- **internal/prompt/** — Prompts interactifs
- **internal/template/** — Moteur de rendu de templates
- **internal/ports/** — Vérification disponibilité ports pour projets générés
- **internal/dbinspect/** — Inspection BDD (migrations, schéma)
- **internal/forge/** — Validation signature projet, métadonnées, suivi features (.forge/forge.yaml, .forge/features.yaml)
- **pkg/generator/** — Types partagés du générateur

Note : `internal/doctor`, `internal/analyze`, `internal/arch`, `internal/project` sont des dossiers vides ; `internal/check` n'existe pas ; la logique se trouve dans `cli/commands.go` et `rules/`.

## Dépendances
- `github.com/spf13/cobra` — Framework CLI
- `gopkg.in/yaml.v3` — Parsing YAML config
- `github.com/jackc/pgx/v5` — Driver PostgreSQL (projets générés)
- Go 1.26 (selon go.mod), CI utilise 1.22 (voir `.github/workflows/ci.yml`)

## Stack du projet généré (figée en V0.1)
- Go stdlib `net/http` avec **Chi router** (`github.com/go-chi/chi/v5`)
- PostgreSQL avec `database/sql` — pas de GORM
- Docker Compose pour le dev local
- Config environnement via `.env` (pattern caarlos0/env)
- Tests table-driven + testcontainers-go optionnel

## Graphe de dépendances des features
```
auth (pas de deps)
  └── cors (dépend de auth)
  └── logging (dépend de auth)
      └── swagger (dépend de cors)
```

## Tâches courantes

### Ajouter une nouvelle commande CLI
1. Ajouter la commande dans `internal/cli/commands.go`
2. Enregistrer dans `internal/cli/root.go` (dans `NewRootCommand`)
3. Ajouter les tests dans `internal/cli/commands_test.go`

### Modifier les templates de projet généré
Éditer les fichiers dans `internal/template/api/` — ce sont des fichiers Go text/template.

### Ajouter une nouvelle feature `forge add` / `forge remove`
1. Implémenter l'interface `Feature` dans `internal/feature/<nom>/`
2. Ajouter les fichiers template dans `internal/template/api/internal/<nom>/`
3. Enregistrer la feature dans `internal/cli/commands.go` dans `newAddCommand()` et `newRemoveCommand()` (voir `auth.AuthFeature{}, cors.CorsFeature{}, logging.LoggingFeature{}, swagger.SwaggerFeature{}`)

## Pipeline CI (`.github/workflows/ci.yml`)
S'exécute sur push/PR vers main/master (utilise Go 1.22) :
1. `go mod download`
2. `go vet ./...`
3. `go test ./...`
4. `go build -o forge ./cmd/forge`
5. Test d'intégration : `forge init` + `go test ./...` dans le projet généré

## Vérifications pre-commit / contribution
```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

## Packages sans tests
Ces packages internes n'ont actuellement pas de fichiers de test :
- `cmd/forge` (point d'entrée, normal)
- `internal/analyzer`
- `internal/app`
- `internal/config`
- `internal/engine`
- `internal/errs`
- `internal/prompt`
- `internal/template`
- `pkg/generator`

## Environnement
- Nécessite Docker pour le workflow Docker du projet généré
- Utilise `.env.example` comme template pour les projets générés
- Port API par défaut : 8080
- Config utilisateur à `~/.forgekit/config.yaml`
- Config projet à `forge.yaml` dans les projets générés

## Processus de release
- Utilise GoReleaser (`.goreleaser.yaml`)
- Build pour linux/amd64 et linux/arm64
- Crée archives .tar.gz, packages .deb, et checksums SHA256
- Version injectée via ldflags : `github.com/Demetrius-ch/forgekit/internal/app.Version`
- Release déclenchée par tags git (v*)

## Références utiles
- `README.md` — Docs utilisateur, exemples d'usage
- `projects.md` — Vision produit et roadmap
- `explication.md` — Analyse concurrentielle et positionnement
- `.github/workflows/ci.yml` — Commandes CI autoritatives
- `Makefile` — Commandes de développement
- `.goreleaser.yaml` — Configuration release