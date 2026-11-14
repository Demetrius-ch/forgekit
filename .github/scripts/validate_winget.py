#!/usr/bin/env python3
"""Validate a WinGet multi-file manifest directory before submitting it.

Checks what microsoft/winget-pkgs rejected in earlier ForgeKit submissions:
exactly one version, one installer and one defaultLocale manifest (no
deprecated singleton), the yaml-language-server schema header, the official
JSON schemas, unknown fields, identifier/version/path consistency and, with
--check-urls, that every InstallerUrl is downloadable, matches its
InstallerSha256 and contains the NestedInstallerFiles.

Usage: validate_winget.py [--check-urls] [--schemas DIR] MANIFEST_DIR
Requires: pip install pyyaml jsonschema
"""

import argparse
import hashlib
import io
import json
import os
import sys
import urllib.request
import zipfile

import jsonschema
import yaml

SCHEMA_URL = "https://raw.githubusercontent.com/microsoft/winget-cli/master/schemas/JSON/manifests/v{v}/manifest.{t}.{v}.json"
HEADER = "# yaml-language-server: $schema=https://aka.ms/winget-manifest.{t}.{v}.schema.json"
REQUIRED_TYPES = ("version", "installer", "defaultLocale")


class Loader(yaml.SafeLoader):
    """YAML loader that keeps dates as strings, as WinGet does."""


Loader.yaml_implicit_resolvers = {
    key: [r for r in resolvers if r[0] != "tag:yaml.org,2002:timestamp"]
    for key, resolvers in yaml.SafeLoader.yaml_implicit_resolvers.items()
}


def load_schema(kind, version, schemas_dir):
    name = f"manifest.{kind}.{version}.json"
    if schemas_dir and os.path.exists(os.path.join(schemas_dir, name)):
        with open(os.path.join(schemas_dir, name), encoding="utf-8") as f:
            return json.load(f)
    with urllib.request.urlopen(SCHEMA_URL.format(v=version, t=kind), timeout=60) as resp:
        return json.load(resp)


def unknown_fields(doc, schema):
    """Fields the schema does not declare; winget validate rejects them."""
    errors = [f"unknown field {k!r}" for k in doc if k not in schema.get("properties", {})]
    installer_def = schema.get("definitions", {}).get("Installer", {}).get("properties")
    if installer_def is not None:
        for i, installer in enumerate(doc.get("Installers") or []):
            errors += [f"Installers[{i}]: unknown field {k!r}" for k in installer if k not in installer_def]
    return errors


def check_installer_urls(doc, root_nested):
    errors = []
    for installer in doc.get("Installers") or []:
        url = installer.get("InstallerUrl")
        try:
            with urllib.request.urlopen(url, timeout=120) as resp:
                data = resp.read()
        except Exception as exc:  # noqa: BLE001 - report any download failure
            errors.append(f"{url}: download failed: {exc}")
            continue
        digest = hashlib.sha256(data).hexdigest()
        if digest.lower() != str(installer.get("InstallerSha256", "")).lower():
            errors.append(f"{url}: SHA256 is {digest}, manifest says {installer.get('InstallerSha256')}")
        nested = installer.get("NestedInstallerFiles") or root_nested or []
        if nested:
            names = set(zipfile.ZipFile(io.BytesIO(data)).namelist())
            for entry in nested:
                path = entry.get("RelativeFilePath", "").replace("\\", "/")
                if path not in names:
                    errors.append(f"{url}: {path} not found in archive")
    return errors


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest_dir")
    parser.add_argument("--check-urls", action="store_true")
    parser.add_argument("--schemas", help="directory holding manifest.<type>.<version>.json files")
    args = parser.parse_args()

    errors = []
    docs = {}
    files = sorted(f for f in os.listdir(args.manifest_dir) if f.endswith(".yaml"))
    for name in files:
        path = os.path.join(args.manifest_dir, name)
        with open(path, encoding="utf-8") as f:
            text = f.read()
        doc = yaml.load(text, Loader=Loader)
        kind, version = doc.get("ManifestType"), doc.get("ManifestVersion")
        if kind not in REQUIRED_TYPES + ("locale",):
            errors.append(f"{name}: ManifestType {kind!r} is not accepted (singleton manifests are deprecated)")
            continue
        if kind in docs:
            errors.append(f"{name}: second {kind} manifest")
        docs[kind] = (name, doc)
        if text.splitlines()[0].strip() != HEADER.format(t=kind, v=version):
            errors.append(f"{name}: first line must be {HEADER.format(t=kind, v=version)}")
        schema = load_schema(kind, version, args.schemas)
        errors += [f"{name}: {e.message}" for e in jsonschema.Draft7Validator(schema).iter_errors(doc)]
        errors += [f"{name}: {e}" for e in unknown_fields(doc, schema)]

    for kind in REQUIRED_TYPES:
        if kind not in docs:
            errors.append(f"missing {kind} manifest")

    if not errors:
        ids = {doc["PackageIdentifier"] for _, doc in docs.values()}
        versions = {str(doc["PackageVersion"]) for _, doc in docs.values()}
        if len(ids) != 1 or len(versions) != 1:
            errors.append(f"PackageIdentifier/PackageVersion differ between files: {ids} {versions}")
        else:
            package_id, package_version = ids.pop(), versions.pop()
            publisher, name = package_id.split(".", 1)
            expected = os.path.join("manifests", publisher[0].lower(), publisher, name, package_version)
            if not os.path.normpath(args.manifest_dir).endswith(expected):
                errors.append(f"directory must end with {expected}")
            for kind, suffix in (("version", ""), ("installer", ".installer")):
                if docs[kind][0] != f"{package_id}{suffix}.yaml":
                    errors.append(f"{docs[kind][0]}: expected file name {package_id}{suffix}.yaml")
            locale = docs["defaultLocale"][1]
            if docs["version"][1].get("DefaultLocale") != locale.get("PackageLocale"):
                errors.append("DefaultLocale of the version manifest must match PackageLocale of the defaultLocale manifest")
            if args.check_urls:
                installer = docs["installer"][1]
                errors += check_installer_urls(installer, installer.get("NestedInstallerFiles"))

    for e in errors:
        print(f"ERROR {e}")
    print("OK" if not errors else f"{len(errors)} error(s)")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
