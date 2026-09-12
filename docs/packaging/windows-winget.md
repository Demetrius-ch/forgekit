# Préparation WinGet pour ForgeKit

Ce document décrit la structure du manifeste WinGet multi-fichiers et le processus de soumission future.

## Structure du Manifeste

Le manifeste est maintenant **multi-fichiers** (schéma WinGet 1.12.0+) dans `packaging/winget/manifests/d/Demetrius-ch/ForgeKit/0.3.6/` :

```
packaging/winget/manifests/d/Demetrius-ch/ForgeKit/0.3.6/
├── Demetrius-ch.ForgeKit.yaml              # Version manifest
├── Demetrius-ch.ForgeKit.installer.yaml    # Installer manifest
└── Demetrius-ch.ForgeKit.locale.en-US.yaml # Locale manifest (en-US)
```

### Artefacts Windows Réels (Release v0.3.6)

| Architecture | Fichier | URL | SHA256 |
|-------------|---------|-----|--------|
| x64 | `forgekit_0.3.6_windows_amd64.zip` | `https://github.com/Demetrius-ch/forgekit/releases/download/v0.3.6/forgekit_0.3.6_windows_amd64.zip` | `e4c74e2e5c23477dee30fa7934809293b0f4c325b59930a16f66ecc79b1cca7e` |
| arm64 | `forgekit_0.3.6_windows_arm64.zip` | `https://github.com/Demetrius-ch/forgekit/releases/download/v0.3.6/forgekit_0.3.6_windows_arm64.zip` | `ac1bd1e4c30cd3b292b11be7afc47283d1ce8fea36018cae73f466200731f785` |

**Note** : Le nommage GoReleaser utilise `windows_amd64` / `windows_arm64` (minuscules), pas `Windows_x86_64` / `Windows_arm64`.

## Validation Locale

```bash
# Installer wingetcreate (nécessite Windows)
wingetcreate validate packaging/winget/manifests/d/Demetrius-ch/ForgeKit/0.3.6/

# Ou valider un fichier spécifique
wingetcreate validate packaging/winget/manifests/d/Demetrius-ch/ForgeKit/0.3.6/Demetrius-ch.ForgeKit.yaml
```

## Processus de Soumission

1. **Préparer la release** : Créer un tag Git `vX.Y.Z` → déclenche GoReleaser
2. **Récupérer les artefacts** : Télécharger les ZIP Windows depuis GitHub Releases
3. **Calculer SHA256** :
   ```bash
   sha256sum forgekit_X.Y.Z_windows_amd64.zip
   sha256sum forgekit_X.Y.Z_windows_arm64.zip
   ```
4. **Mettre à jour les 3 manifestes** dans `packaging/winget/manifests/d/Demetrius-ch/ForgeKit/X.Y.Z/` :
   - `Demetrius-ch.ForgeKit.yaml` : version, release date
   - `Demetrius-ch.ForgeKit.installer.yaml` : URLs, SHA256
   - `Demetrius-ch.ForgeKit.locale.en-US.yaml` : version (optionnel)
5. **Valider** : `wingetcreate validate packaging/winget/manifests/d/Demetrius-ch/ForgeKit/X.Y.Z/`
6. **Forker** : `microsoft/winget-pkgs`
7. **Créer la structure** dans le fork :
   ```
   manifests/d/Demetrius-ch/ForgeKit/X.Y.Z/
   ├── Demetrius-ch.ForgeKit.yaml
   ├── Demetrius-ch.ForgeKit.installer.yaml
   └── Demetrius-ch.ForgeKit.locale.en-US.yaml
   ```
8. **Pull Request** : Soumettre vers `microsoft/winget-pkgs:master`

## Exigences WinGet (Schema 1.12.0)

- `PackageIdentifier` : `Demetrius-ch.ForgeKit` (format Publisher.Name)
- `PackageVersion` : Version exacte du tag
- `InstallerType` : `zip` (archive portable)
- `NestedInstallerType` : `portable` (binaire autonome)
- `NestedInstallerFiles.RelativeFilePath` : `forgekit.exe` (à la racine du ZIP)
- `PortableCommandAlias` : `forgekit` (commande exposée dans PATH)
- `Architecture` : `x64` et `arm64` (pas `x86` ni `arm`)
- `ManifestVersion` : `1.12.0` (dernière stable)

## Notes Importantes

- **Ne pas soumettre avant** que les binaires Windows soient publiés sur GitHub Releases
- **Checksums obligatoires** : WinGet rejette les manifestes sans SHA256 valides
- **URLs d'installateur** : Doivent pointer vers `github.com/Demetrius-ch/forgekit/releases/download/v...`
- **Signature Authenticode** : Recommandée pour éviter SmartScreen, mais pas obligatoire pour la soumission initiale
- **Publisher** : Doit correspondre à l'identité GitHub (`Demetrius-ch`)

## Commandes Exécutées Après Installation

WinGet ajoute automatiquement le dossier d'installation au PATH utilisateur. Les commandes disponibles :

```powershell
forgekit version
forgekit init mon-api
forgekit doctor
forgekit add auth
forgekit --help
```

L'alias `forge` est aussi disponible via le même binaire.

## Mises à Jour Futures

Pour chaque nouvelle version :
1. Tag Git → GoReleaser publie les artefacts
2. Mettre à jour les 3 manifestes : version, URLs, SHA256, ReleaseDate
3. Soumettre PR vers `microsoft/winget-pkgs:master`

## Ressources

- [WinGet Manifest Specification](https://github.com/microsoft/winget-pkgs/blob/master/manifest-spec/manifest-schema.md)
- [wingetcreate Tool](https://github.com/microsoft/wingetcreate)
- [Submission Process](https://github.com/microsoft/winget-pkgs/blob/master/CONTRIBUTING.md)
- [ForgeKit Releases](https://github.com/Demetrius-ch/forgekit/releases)