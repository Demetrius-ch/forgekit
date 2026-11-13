# Publication de ForgeKit sur WinGet

Ce document décrit le manifeste WinGet de ForgeKit et la procédure pour le faire
accepter dans [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).

## État actuel

ForgeKit n'est **pas encore** dans le dépôt WinGet. Les deux soumissions de la v0.3.6
ont été fermées sans être fusionnées :

| PR | Problème signalé | Statut |
|----|------------------|--------|
| [#433737](https://github.com/microsoft/winget-pkgs/pull/433737) | Manifeste *singleton* (format déprécié) | Fermée par l'auteur |
| [#433812](https://github.com/microsoft/winget-pkgs/pull/433812) | Singleton encore présent (avec une URL `..._Windows_x86_64.zip` qui renvoie 404), locale de type `locale` au lieu de `defaultLocale`, en-têtes de schéma manquants | Fermée automatiquement : pas de réponse dans les 3 jours suivant la demande de modifications |

Le CLA Microsoft est déjà signé pour le compte `Demetrius-ch` : il ne sera pas redemandé.

Les manifestes de `packaging/winget/` ont été corrigés et sont vérifiés par
`.github/workflows/winget.yml` à chaque modification.

## Structure attendue

Exactement **trois** fichiers, pas de manifeste singleton :

```
manifests/d/Demetrius-ch/ForgeKit/0.3.6/
├── Demetrius-ch.ForgeKit.yaml               # ManifestType: version (DefaultLocale: en-US)
├── Demetrius-ch.ForgeKit.installer.yaml     # ManifestType: installer
└── Demetrius-ch.ForgeKit.locale.en-US.yaml  # ManifestType: defaultLocale
```

Règles vérifiées par le validateur, et à l'origine des refus passés :

- la première ligne de chaque fichier est l'en-tête de schéma, par exemple
  `# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.12.0.schema.json` ;
- le manifeste de version contient seulement `PackageIdentifier`, `PackageVersion`,
  `DefaultLocale`, `ManifestType`, `ManifestVersion` ;
- la locale par défaut (`en-US`) est de type `defaultLocale` ;
- aucun champ inconnu (`Commands` va dans le manifeste installer, pas dans la locale) ;
- `InstallerSwitches` absent pour un zip portable (des valeurs vides sont refusées) ;
- les URLs existent, leur SHA256 correspond et l'archive contient `forgekit.exe`.

La commande installée est `forgekit` (alias portable). Il n'y a pas d'alias `forge` sous Windows.

## Valider avant de soumettre

Sous Linux, macOS ou Windows (Python 3) :

```bash
pip install pyyaml jsonschema
python .github/scripts/validate_winget.py --check-urls packaging/winget/manifests/d/Demetrius-ch/ForgeKit/0.3.6
```

Sous Windows, avec le client officiel, en plus :

```powershell
winget validate --manifest packaging\winget\manifests\d\Demetrius-ch\ForgeKit\0.3.6
# Test d'installation local (nécessite: winget settings --enable LocalManifestFiles, en administrateur)
winget install --manifest packaging\winget\manifests\d\Demetrius-ch\ForgeKit\0.3.6
forgekit version
```

## Soumettre

1. Forker `microsoft/winget-pkgs` (ou synchroniser le fork existant avec `master`).
2. Créer une branche, par exemple `Demetrius-ch.ForgeKit-0.3.6`.
3. Copier **uniquement** les trois fichiers dans `manifests/d/Demetrius-ch/ForgeKit/0.3.6/`.
   Vérifier qu'aucun autre fichier (ancien singleton) ne reste dans ce dossier.
4. Ouvrir la PR vers `microsoft/winget-pkgs:master`, titre : `New package: Demetrius-ch.ForgeKit version 0.3.6`.
5. Suivre la PR : répondre aux commentaires des bots et des relecteurs **dans les 3 jours**,
   sinon la PR est fermée automatiquement (cas de #433812). Pousser les corrections sur la même branche.

Alternative outillée (Windows) : `wingetcreate submit --token <PAT> packaging\winget\manifests\d\Demetrius-ch\ForgeKit\0.3.6`
crée le fork, la branche et la PR automatiquement.

## Nouvelle version

1. Tag `vX.Y.Z` → GoReleaser publie `forgekit_X.Y.Z_windows_amd64.zip` et `forgekit_X.Y.Z_windows_arm64.zip`.
2. Copier le dossier `0.3.6` en `X.Y.Z` et mettre à jour `PackageVersion`, `ReleaseDate`,
   `ReleaseNotesUrl`, les URLs et les SHA256 (`sha256sum forgekit_X.Y.Z_windows_*.zip`).
3. Lancer le validateur (la CI le fait aussi), puis soumettre comme ci-dessus.
   Une fois le paquet accepté, `wingetcreate update Demetrius-ch.ForgeKit --version X.Y.Z --urls <x64> <arm64> --submit`
   automatise ces étapes.

## Références

- [Spécification des manifestes](https://github.com/microsoft/winget-pkgs/blob/master/doc/manifest/README.md)
- [Guide de contribution winget-pkgs](https://github.com/microsoft/winget-pkgs/blob/master/CONTRIBUTING.md)
- [wingetcreate](https://github.com/microsoft/winget-create)
