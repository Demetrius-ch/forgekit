# Audit de stabilisation ForgeKit — 2026-10-30

Base auditée : `main` @ `064c5cf` + modifications locales non commitées. Go 1.27.3, linux/amd64.
Aucun commit, tag, push, release ni publication n'a été effectué.

## 0. État de l'arbre de travail trouvé

| Élément | Constat | Action |
|---|---|---|
| `internal/cli/commands.go` (modifié, 6–13 oct.) | WIP v0.8 : −2 500 lignes ; `init/add/remove/inspect/doctor/analyze/check/config` remplacés par des stubs « non implémentée » ; ajout `status`, `update --check`, `diff`, `history`, `rollback` | Sauvegardé dans `_wip-v0.8/`, restauré depuis HEAD (accord utilisateur) |
| `internal/cli/root.go`, `status.go`, `commands.go.tmp` | WIP v0.8 | Sauvegardés dans `_wip-v0.8/`, retirés du build |
| `internal/diff/`, `internal/migration/` | WIP v0.8, compilent, non utilisés | Laissés en place |
| `forge` (binaire suivi par git malgré `.gitignore`) | Modifié | Non touché |
| `go.sum`, `.gitignore`, `11.md`, `test-api/` | Modifs locales | Non touchés |

Conséquence avant restauration : `go build ./...` et `go vet ./...` échouaient
(`internal/errs` → `cli.CiError` indéfini ; `commands_test.go` → `runReport` indéfini).

## 1. Architecture réelle

```
cmd/forge/main.go ── cli.NewRootCommand() ── errs.ExitCode(err)
        │
internal/cli ─────────────┬─ init ──► generator ──► generationplan (registre de composants par langage)
 (cobra, flags globaux     │              │             └─► template (embed api/v04/**, text/template)
  --format/--quiet/--debug)│              ├─► engine (exécution du plan, dry-run)
                           │              ├─► forge (métadonnées .forge/forge.yaml)
                           │              └─► go mod / npm install / python venv+pip (exec, sans shell)
                           │     ├─► ports (détection de ports libres)
                           │     ├─► dbinspect (pgx : bases PostgreSQL existantes, migrations)
                           │     ├─► prompt (questions interactives)
                           │     └─► projectconfig (configuration typée + validation)
                           ├─ add/remove ──► feature (registre, dépendances, détection, intégration
                           │                  router.go/main.go, .env.example, features.yaml)
                           │                  └─► feature/{auth,cors,logging,swagger}
                           │                  + snapshot/rollback (dans cli/commands.go)
                           ├─ doctor/analyze/check ──► rules (architecture, sécurité, env, docker)
                           │                            └─► report (findings, score) ─► output
                           ├─ inspect ──► forge.ValidateSignature
                           ├─ config init ──► écrit forge.yaml
                           └─ version, completion (cobra)
internal/config      : forge.yaml projet + ~/.config/forge/config.yaml (auteur)
internal/analyzer    : non référencé par le CLI
pkg/generator        : façade publique
internal/errs        : codes de sortie (dépend de cli.CiError)
```

Composants **absents** (signalés car listés dans la mission) : registre de templates,
templates externes, commandes `forge template list|info|install|remove|validate|check`.
`templateVersion` existe dans la configuration mais reste toujours vide.

Deux mécanismes de rollback coexistent : le snapshot du CLI (`cli/commands.go`, utilisé) et
`feature.RollbackManager` (`feature/rollback.go`, utilisé uniquement par le WIP v0.8).

## 2. Commandes réellement disponibles

| Commande | Lit | Écrit | Tests |
|---|---|---|---|
| `init <nom>` (+ `--module --port --postgres-port --db-name --author --dir --non-interactive --dry-run --skip-postprocess --language --architecture --database --docker --auth --docs --tests --ci --config`) | `--config`, `~/.config/forge/config.yaml`, ports locaux, PostgreSQL localhost (sondage) | répertoire projet, `.forge/forge.yaml`, `go.mod/go.sum`, `node_modules`, `.venv` | generator (unitaires), e2e |
| `add [feature]` (`--list --plan --dry-run`) | go.mod, router.go, main.go, `.forge/` | fichiers de la feature, go.mod/go.sum, `.env.example`, router.go, main.go, `.forge/features.yaml` | feature/*, cli, e2e |
| `remove <feature>` (`--plan`) | idem | idem (suppression) | e2e |
| `doctor` (`--ci`) | projet, Docker, outils | rien | rules, cli |
| `analyze` (`--ci`) | projet, forge.yaml | rien | cli |
| `check` | projet, forge.yaml | rien | rules |
| `inspect` | `.forge/` | rien | forge |
| `config init` | — | `forge.yaml` | aucun |
| `version`, `completion` | — | — | cli (branding) |

Écarts documentation ↔ code :
- `AGENTS.md` classe `doctor`, `analyze`, `check` comme « modifiant l'état » : ils sont en lecture seule.
- `AGENTS.md` : « en cas de conflit, l'installation s'arrête et attend confirmation » → faux : les conflits sont seulement affichés dans `--plan`, l'installation écrase sans confirmation.
- `README.md` (« Generated project ») décrit encore la structure v0.3 (`domain/user.go`, `application/user`, `postgres/pool.go`) ; la génération v0.4 ne produit que l'exemple `health`.
- `app.Version = "0.4.0"` alors que le code inclut le support Python (v0.6) ; les releases l'écrasent via ldflags, mais les builds locaux et `.forge/forge.yaml` affichent 0.4.0.
- `README` annonce Go 1.22+, `go.mod` exige `go 1.26`.

## 3. Classification des fonctionnalités (vérifiée par exécution)

| Fonctionnalité | Avant | Après correctifs |
|---|---|---|
| Configuration (flags, `--config`, interactif) | Partiellement défectueuse (réponses scriptées perdues, ports incohérents si la base change) | Implémentée et testée |
| Architectures Go hexagonal/clean/layered | Implémentée et testée | idem |
| Bases postgres/mysql/sqlite/none (Go) | Implémentée et testée | idem |
| Docker (Go) | Implémentée et testée | idem |
| Docker (TS/JS/Python) | Présente mais défectueuse (TS lance `.ts` sans build ; port interne ≠ port publié ; `DATABASE_URL` au format Go) | Corrigée, testée par rendu — **pas de `docker build` réel** (démon indisponible) |
| Auth JWT (Go) | Implémentée et testée | idem |
| Auth JWT (TS/JS/Python) | Présente mais défectueuse (`jsonwebtoken`, `jose`, `passlib` non déclarés) | Corrigée, vérifiée (tsc, import Node, import Python) |
| Swagger (Go) | Implémentée et testée | idem |
| Swagger (TS/JS) | Présente mais défectueuse (`init` échoue : « duplicate destination package.json ») | Refus explicite (non implémentée) |
| Swagger (Python) | Présente mais défectueuse (même erreur) | Fonctionne (FastAPI sert `/docs` nativement) |
| Tests générés Go | Implémentée et testée | idem |
| Tests générés TS/JS | Présente mais défectueuse (test appelle `localhost:0` sans serveur : échoue toujours) | Corrigée, `npm test` passe |
| Tests générés Python | Présente mais défectueuse (`pytest`, `httpx` absents) | Corrigée, `pytest` passe |
| Tests `unit-integration` TS/JS/Python | Partiellement implémentée (identique à `unit`) | inchangé |
| CI générée | Go : OK. TS/JS/Python : défectueuse (lance `go vet/test/build`) | Corrigée par langage |
| Langages Go / TS / JS / Python | Go OK ; autres partiellement défectueux | Vérifiés en réel (init + tests) |
| Features `add` sur projet généré par `init` | **Défectueuse** : `cors`, `logging`, `swagger` échouent sur tout projet v0.4 ; échec = projet cassé | Corrigée sur les 3 architectures, compilation + exécution vérifiées |
| `add`/`remove` TS/JS | Message trompeur (« router.go manquant ») | Refus explicite |
| `add`/`remove` Python | Refus explicite | idem |
| Rollback | Défectueux : `.env.example`, fichiers créés, router.go, main.go non restaurés | Corrigé et testé |
| Métadonnées `.forge` | Implémentée et testée | idem |
| Base PostgreSQL existante / externe (interactif) | Partiellement implémentée : les choix (hôte, mot de passe…) sont collectés mais **ignorés** par les templates v0.4 | inchangé (signalé) |
| Autocomplétion | Implémentée (cobra), non testée | idem |
| Options non interactives | Implémentée et testée | idem |
| Sorties colorées / chronométrées | Implémentée ; spinner illisible hors TTY | Spinner corrigé hors TTY ; couleurs ANSI encore émises hors TTY (désactivables par `NO_COLOR`/`CI`) |
| JSON | `init` sans `language` ; `add`/`remove` n'émettaient rien | Corrigé |
| Templates registry / externes / `forge template` | Absents | Absents |

## 4. État initial des tests (avant toute modification)

| Vérification | Arbre de travail | HEAD 064c5cf |
|---|---|---|
| `go build ./...` | **échec** (WIP) | OK |
| `go vet ./...` | **échec** (WIP) | OK |
| `go test ./...` | non compilable | OK (tous les packages) |
| `go test -tags e2e ./internal/e2e/` | — | **4 échecs / 8** (`FullWorkflow`, `ErrorScenarios`, `RemovePlan` : bug d'intégration features ; `ForgeInit` : attentes v0.3) |
| `gofmt -l` | 8 fichiers projet non formatés + 2 fichiers vendor | 7 fichiers projet |

Les tests E2E (tag `e2e`) ne sont exécutés ni par `make test` ni par la CI GitHub.

## 5. Corrections apportées

1. **Features sur projets v0.4** (`feature/integration.go`, `cors.go`, `logging.go`, `auth.go`, `swagger.go`) :
   insertion des middlewares après `chi.NewRouter()` et des routes avant `return r` quand le router n'a pas de pile middleware ;
   middlewares encapsulés dans `r.Use(...)` (l'ancien code insérait une instruction nue qui ne compile pas) ;
   `main.go` n'est plus modifié avec un import inutilisé ; `Check` accepte les routers clean/layered.
2. **Rollback** (`cli/commands.go`) : le snapshot couvre `.env.example`, router, `main.go` et les fichiers des plans ; les fichiers créés et répertoires vides sont supprimés.
3. **`.env.example`** : plus de variables dupliquées lors d'installations successives.
4. **`--ci`** : un succès ne renvoie plus une erreur vide (`erreur : ` sur stderr).
5. **`--format`** : valeur invalide refusée (avant : ignorée silencieusement).
6. **Spinner** : pas d'animation quand la sortie n'est pas un terminal.
7. **Prompts** : un seul lecteur stdin partagé (les réponses redirigées étaient perdues après la première).
8. **`init`** : `--database` prime sur `--config` aussi pour le choix des ports ; ports re-sélectionnés si la base change en interactif ; module validé selon le langage ; JSON avec `language` et sans ligne vide ; étapes suivantes Python ; libellé « Package » hors Go ; affichage dry-run du port demandé.
9. **`add`/`remove`** : résultat JSON ; refus explicite pour TS/JS.
10. **Templates** : CI par langage ; Dockerfile TS compilé ; compose (`PORT`, `DATABASE_URL` par langage) ; dépendances Python/Node réellement importées ; URLs de base cohérentes avec Docker ; tests TS/JS fonctionnels ; Swagger Python/TS/JS.
11. **`make lint`** : ne réécrit plus `vendor/`.
12. **Test E2E `TestForgeInitE2E`** : liste de fichiers mise à jour vers la structure v0.4 documentée (aucune autre vérification retirée).

Tests ajoutés : `feature/integration_test.go`, `cli/rollback_test.go`, `generator/language_components_test.go`,
`output/spinner_test.go` (cas non-TTY), `e2e/init_e2e_test.go` (réponses interactives redirigées).

## 6. État final des vérifications

`make lint` (vendor intact) · `make vet` OK · `go build ./...` OK · `go test ./...` OK ·
`go test -tags e2e ./internal/e2e/` **9/9 OK** · compilation croisée windows/amd64, windows/arm64, darwin/arm64, linux/arm64 OK.
Vérifications manuelles : init réel Go (3 architectures + add/remove de toutes les features + router exécuté via httptest),
TypeScript (tsc, npm test, build), JavaScript (npm test), Python postgres/mysql (imports, pytest).

Non vérifiable ici : `docker build` / `docker compose up` (démon Docker indisponible), CI GitHub Actions générée, Windows/macOS à l'exécution.

## 7. Problèmes restants (non corrigés, à arbitrer)

- Conflits `add` : écrasement sans confirmation (doc AGENTS.md inexacte).
- Choix « base existante / serveur externe » collectés mais ignorés par les templates v0.4 ; le mot de passe saisi n'est écrit nulle part (les anciens templates v0.3 l'écrivaient dans `.env.example`).
- `init` non interactif sonde `localhost:5432` avec `postgres:postgres` (effet réseau implicite).
- Le volume SQLite `/data` des conteneurs est probablement non inscriptible par l'utilisateur `app` (non vérifié faute de Docker).
- Le mode `unit-integration` ne génère pas de test d'intégration pour TS/JS/Python.
- Couleurs ANSI émises hors TTY ; `--skip-postprocess` ne saute pas `npm install` / `pip install` (réseau requis).
- `internal/analyzer` non utilisé ; deux mécanismes de rollback.
- Templates v0.3 (`internal/template/api/*` hors `v04`) encore utilisés par les features, ce qui couple les features à l'ancienne structure.
- Version `0.4.0` en dur ; README « Generated project » obsolète ; binaire `forge` suivi par git.

## 8. Préparation v0.8

Le WIP sauvegardé dans `_wip-v0.8/` part d'un `commands.go` qui supprime les commandes existantes :
pour le reprendre, il faudra ajouter les nouvelles commandes à côté de l'implémentation actuelle
(et non la remplacer), et choisir un seul mécanisme de rollback/snapshot entre `cli` et `feature.RollbackManager`.
