# Migration Guide: v0.3 → v0.4

## What Changed

ForgeKit v0.4 introduces customizable project generation while maintaining full backward compatibility with v0.3 projects.

### Key Changes

| Aspect | v0.3 | v0.4 |
|--------|------|------|
| Architecture | Hexagonal only | Hexagonal, Clean, Layered |
| Database | PostgreSQL only | PostgreSQL, MySQL, SQLite, None |
| Docker | Always included | Optional (on/off) |
| Auth | None by default | None, JWT |
| Docs | None by default | None, Swagger |
| Init mode | Interactive only | Interactive + Non-interactive |
| Config | Single format | `forge.yaml` with architecture/database fields |

### Backward Compatibility

- **Existing v0.3 projects continue to work** — `forgekit doctor`, `analyze`, `check`, `inspect` work on v0.3 projects
- **`.forge` system is compatible** — v0.4 can read and validate v0.3 project signatures
- **`forgekit add`/`remove` still works** — Feature system is unchanged, adaptive paths detect architecture automatically
- **No breaking changes to CLI** — All existing flags and commands remain valid

### New Capabilities

#### Custom Project Generation

```bash
# Interactive mode — prompts for all options
forgekit init my-api

# Non-interactive mode — specify everything via flags
forgekit init my-api \
  --module github.com/example/my-api \
  --architecture hexagonal \
  --database postgres \
  --docker \
  --auth jwt \
  --docs swagger \
  --tests unit+integration \
  --ci github \
  --non-interactive

# Preview without creating files
forgekit init my-api --dry-run

# JSON output for automation
forgekit init my-api --format json
```

#### Supported Configurations

| Architecture | Database | Docker | Auth | Docs | Tests | CI |
|-------------|----------|--------|------|------|-------|----|
| Hexagonal | PostgreSQL | Yes/No | None/JWT | None/Swagger | Unit/Integration | None/GitHub |
| Clean | MySQL | Yes/No | None/JWT | None/Swagger | Unit/Integration | None/GitHub |
| Layered | SQLite | Yes/No | None/JWT | None/Swagger | Unit/Integration | None/GitHub |
| Layered | None | Yes/No | None/JWT | None/Swagger | Unit/Integration | None/GitHub |

### Migration Steps

#### For New Projects

No migration needed — use `forgekit init` with the new flags.

#### For Existing v0.3 Projects

1. **No action required** — v0.3 projects continue to work as-is
2. **Optional**: Update `forge.yaml` to include new fields if you want to use v0.4 features
3. **Run `forgekit doctor`** to verify project health
4. **Add features as needed**: `forgekit add auth`, `forgekit add swagger`, etc.

### Files Changed in v0.4

- `forge.yaml` — Added `architecture`, `database`, `docker`, `authentication`, `documentation`, `tests`, `ci` fields
- Generated `main.go` — Graceful shutdown with `signal.Notify` + `server.Shutdown`
- Feature integration — Auto-detects router/main paths across architectures

### Files Unchanged in v0.4

- Feature system (`internal/feature/`) — No changes
- `.forge` system — No changes
- CLI commands — No breaking changes
- Existing templates — v0.3 templates remain available

### Troubleshooting

**Problem**: `forgekit doctor` reports unknown architecture
**Solution**: v0.3 projects don't have `architecture` field in `forge.yaml`. This is expected — `doctor` handles it gracefully.

**Problem**: `forgekit add` can't find router path
**Solution**: Feature integration now auto-detects paths across all architectures. Update ForgeKit if this was an issue in v0.3.

**Problem**: Generated project has different structure than v0.3
**Solution**: This is expected — v0.4 generates architecture-specific paths. Use `forgekit init --architecture hexagonal` for the same structure as v0.3.

### Performance

v0.4 has no performance regression compared to v0.3. Benchmarks show comparable generation times across architectures.

### Support

For issues or questions, open an issue on GitHub.
