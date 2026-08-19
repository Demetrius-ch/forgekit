# Contribuer à ForgeKit

Merci de votre intérêt pour contribuer à ForgeKit !

## Démarrage rapide

```bash
# Fork & clone
git clone https://github.com/VOTRE_USERNAME/forgekit.git
cd forgekit

# Installer les dépendances
go mod download

# Lancer les tests
go test ./...

# Build
go build -o forge ./cmd/forge
```

## Types de contributions

| Type                   | Description                                     |
| ---------------------- | ----------------------------------------------- |
| **Bug fixes**          | Corrections de bugs existants                   |
| **Nouvelles features** | Nouvelles commandes, features `forge add`, etc. |
| **Documentation**      | README, guides, commentaires code               |
| **Tests**              | Tests unitaires, d'intégration, E2E             |
| **Refactoring**        | Amélioration code sans changement fonctionnel   |
| **VS Code Extension**  | Améliorations extension VS Code                 |

## Checklist avant PR

- [ ] Tests passent : `go test ./...`
- [ ] Code formaté : `gofmt -w .`
- [ ] Pas d'erreurs vet : `go vet ./...`
- [ ] Build réussi : `go build ./cmd/forge`
- [ ] Tests E2E : `go test -tags=e2e ./internal/e2e/...`
- [ ] Messages de commit clairs (format conventional commits)
- [ ] Documentation mise à jour si nécessaire

##  Format des commits

```
type(scope): description courte

Description plus longue si nécessaire.

Fixes #123
```

**Types** : `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `perf`

**Exemples** :

```
feat(generator): add support for custom templates
fix(cli): handle missing go.mod gracefully
docs(readme): update installation instructions
test(generator): add e2e test for custom module path
```

## Lancer les tests

```bash
# Tous les tests
go test ./...

# Tests spécifiques
go test ./internal/generator/...
go test ./internal/feature/...

# Tests E2E (nécessite Docker)
go test -tags=e2e ./internal/e2e/...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Ajouter une nouvelle feature `forge add`

1. Créer le dossier : `internal/feature/mafeature/`
2. Implémenter l'interface `Feature` :
   - `mafeature.go` : logique principale
   - `mafeature_test.go` : tests
3. Ajouter templates dans `internal/template/api/internal/mafeature/`
4. Enregistrer dans `internal/cli/commands.go` dans `newAddCommand()`
5. Tests : `go test ./internal/feature/mafeature/...`

## Extension VS Code

Le code est dans `integrations/vscode/` :

```bash
cd integrations/vscode
npm install
npm run compile
npm run lint
vsce package
```

## Documentation

- `README.md` : Vue d'ensemble projet
- `AGENTS.md` : Instructions pour agents IA
- Commentaires de code : GoDoc sur fonctions publiques

## Signaler un bug

Utilisez le template **Bug Report** sur GitHub Issues avec :

- Version ForgeKit (`forge version`)
- OS / Architecture
- Étapes de reproduction minimales
- Logs complets

## Proposer une feature

Utilisez le template **Feature Request** sur GitHub Issues.

## Sécurité

Pour les vulnérabilités : [Security Advisories](
   
)

## Contact

- Issues GitHub : pour bugs/features
- Discussions GitHub : pour questions/général
- Security Advisories : pour vulnérabilités

---

**Merci de contribuer à ForgeKit !**
