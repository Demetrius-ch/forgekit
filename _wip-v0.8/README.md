# WIP v0.8 sauvegardé (non compilé)

Copie de l'état local non commité au 2026-10-30, avant l'audit de stabilisation.
Ce dossier commence par `_` : il est ignoré par `go build ./...` et `go test ./...`.

- `internal/cli/commands.go` : version WIP (commandes status, update --check, diff, history, rollback ;
  init/add/remove/inspect/doctor/analyze/check/config remplacées par des stubs).
- `internal/cli/root.go` : enregistrement des nouvelles commandes.
- `internal/cli/status.go`, `internal/cli/commands.go.tmp` : brouillons.
- `working-tree.diff` : `git diff` complet de l'arbre de travail au moment de la sauvegarde.

`internal/diff/` et `internal/migration/` sont restés en place (ils compilent, non utilisés).
