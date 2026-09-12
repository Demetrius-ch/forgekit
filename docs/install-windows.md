# Installation de ForgeKit sur Windows

## Téléchargement et Installation Manuelle

### 1. Télécharger l'archive Windows

Rendez-vous sur la page [GitHub Releases](https://github.com/Demetrius-ch/forgekit/releases) et téléchargez l'archive correspondant à votre architecture :

- **Windows x86_64 (64-bit)** : `forgekit_<version>_Windows_x86_64.zip`
- **Windows ARM64** : `forgekit_<version>_Windows_arm64.zip`

### 2. Extraire l'archive

Faites un clic droit sur le fichier `.zip` téléchargé → **Extraire tout...** → Choisissez un dossier de destination.

L'archive contient :
- `forgekit.exe` — L'exécutable principal
- `README.md` — Documentation rapide
- `LICENSE` — Licence MIT

### 3. Ajouter ForgeKit au PATH Windows

#### Option A : PATH Utilisateur (Recommandé, ne nécessite pas d'administrateur)

1. Ouvrez **Paramètres** → **Système** → **À propos** → **Paramètres système avancés** → **Variables d'environnement**
2. Dans **Variables utilisateur**, sélectionnez `Path` → **Modifier**
3. Cliquez **Nouveau** et ajoutez le chemin complet du dossier contenant `forgekit.exe` (ex: `C:\Users\VotreNom\forgekit`)
4. Cliquez **OK** sur toutes les fenêtres

#### Option B : PATH Système (Nécessite administrateur)

1. Même procédure mais dans **Variables système**
2. Utile si plusieurs utilisateurs doivent accéder à ForgeKit

### 4. Vérifier l'installation

Ouvrez un **nouveau** PowerShell ou CMD (important : redémarrez le terminal après modification du PATH) :

```powershell
forgekit version
```

Vous devriez voir quelque chose comme :
```
forgekit version 0.3.4
```

### 5. Tester ForgeKit

```powershell
# Diagnostic environnement
forgekit doctor

# Créer un nouveau projet API
forgekit init mon-api
```

Cela crée un dossier `mon-api/` avec un projet Go prêt à l'emploi.

---

## Prérequis pour Utiliser ForgeKit

| Outil | Requis pour | Installation |
|-------|-------------|--------------|
| **Go 1.22+** | `forge init` (génération + `go mod tidy`), `forge add/remove`, `forge doctor` | [go.dev/dl](https://go.dev/dl/) |
| **Git** | `forge doctor`, versioning projets | [git-scm.com](https://git-scm.com/download/win) |
| **Docker Desktop** | Exécuter les projets générés (`docker compose up`), `forge doctor` (check Docker) | [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/) |
| **golangci-lint** | `forge check` (optionnel) | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |

> **Note** : ForgeKit lui-même est un binaire autonome — **pas besoin de Go installé** pour exécuter `forgekit version`, `forgekit doctor`, `forgekit inspect`, etc. Go n'est requis que pour les opérations qui compilent ou testent le code généré.

---

## Utilisation dans PowerShell vs CMD

ForgeKit fonctionne dans les deux :

```powershell
# PowerShell (recommandé)
forgekit version
forgekit init mon-api

# CMD (fonctionne aussi)
forgekit version
forgekit init mon-api
```

Les couleurs et la sortie formatée fonctionnent dans les deux (PowerShell 7+, Windows Terminal, CMD moderne).

---

## Installation via WinGet (Futur)

Une fois le package accepté dans le dépôt officiel WinGet :

```powershell
winget install ForgeKit
```

> Actuellement en préparation. Le manifeste WinGet sera disponible dans `packaging/winget/` du dépôt.

---

## Mise à jour

Téléchargez la nouvelle version depuis GitHub Releases, remplacez l'ancien `forgekit.exe` dans votre dossier PATH.

---

## Désinstallation

1. Supprimez `forgekit.exe` de votre dossier
2. Retirez le dossier du PATH (Variables d'environnement)
3. Redémarrez le terminal

---

## Dépannage

### `forgekit` n'est pas reconnu
- Vérifiez que le dossier contenant `forgekit.exe` est bien dans le PATH
- Redémarrez **complètement** votre terminal (PowerShell/CMD)
- Vérifiez avec `where.exe forgekit`

### Windows SmartScreen / Windows Defender bloque l'exécution
- C'est normal pour un nouveau binaire téléchargé depuis Internet
- Cliquez **Informations complémentaires** → **Exécuter quand même**
- La réputation du binaire s'améliorera avec le temps
- Les checksums SHA256 sont publiés sur GitHub Releases pour vérification

### `forge init` échoue avec "go: command not found"
- Installez Go depuis [go.dev/dl](https://go.dev/dl/)
- Redémarrez le terminal après installation
- Ou utilisez `--skip-postprocess` pour générer sans `go mod tidy` / tests

### Docker non trouvé dans `forge doctor`
- Installez Docker Desktop
- Démarrez Docker Desktop après installation
- `forge init` fonctionne sans Docker — Docker n'est nécessaire que pour **exécuter** le projet généré