# Copybara Infrastructure

This directory contains the Copybara configuration for syncing gazelle extensions from this monorepo to their respective public repositories.

## Architecture

```
bazelle (monorepo)                    Public Repos
├── gazelle-kotlin/    ──push──>      albertocavalcante/gazelle-kotlin
├── gazelle-python/    ──push──>      albertocavalcante/gazelle-python
├── gazelle-groovy/    ──push──>      albertocavalcante/gazelle-groovy
├── internal/log/      (vendored into each)
├── pkg/jvm/           (vendored into kotlin, groovy)
└── pkg/treesitter/    (vendored into kotlin only)
```

## Workflows

### Push (monorepo → public repos)

| Workflow | Source | Destination | Shared Code |
|----------|--------|-------------|-------------|
| `push-kotlin` | `gazelle-kotlin/` | `gazelle-kotlin` | log, jvm, treesitter |
| `push-python` | `gazelle-python/` | `gazelle-python` | log |
| `push-groovy` | `gazelle-groovy/` | `gazelle-groovy` | log, jvm |

### Pull (public repos → monorepo)

| Workflow | Source | Destination |
|----------|--------|-------------|
| `pull-kotlin` | `gazelle-kotlin` | `gazelle-kotlin/` |
| `pull-python` | `gazelle-python` | `gazelle-python/` |
| `pull-groovy` | `gazelle-groovy` | `gazelle-groovy/` |

### Validate (dry-run)

| Workflow | Description |
|----------|-------------|
| `validate-kotlin` | Test Kotlin transformation without pushing |
| `validate-python` | Test Python transformation without pushing |
| `validate-groovy` | Test Groovy transformation without pushing |

## Transformations

Each push workflow:

1. **Extracts** the plugin directory and shared dependencies
2. **Moves** files to the correct structure in the destination
3. **Rewrites** import paths from `github.com/albertocavalcante/bazelle/...` to `github.com/albertocavalcante/gazelle-{plugin}/...`
4. **Rewrites** `go.mod` module path

Commits are tracked with `Bazelle-RevId` label to prevent re-migration.

## Setup Requirements

### 1. Create Destination Repositories

```bash
gh repo create albertocavalcante/gazelle-kotlin --public --description "Gazelle extension for Kotlin (rules_kotlin)"
gh repo create albertocavalcante/gazelle-python --public --description "Gazelle extension for Python (rules_python)"
gh repo create albertocavalcante/gazelle-groovy --public --description "Gazelle extension for Groovy"
```

### 2. Configure Eukia GitHub App

**Eukia** is the GitHub App used for authenticated git operations.

#### Required Permissions

- **Contents**: Read & write
- **Pull Requests**: Read & write
- **Workflows**: Read & write

#### Repository Access

Grant Eukia access to **all 4 repositories**:
- `albertocavalcante/bazelle`
- `albertocavalcante/gazelle-kotlin`
- `albertocavalcante/gazelle-python`
- `albertocavalcante/gazelle-groovy`

### 3. Add Repository Secrets

In the `bazelle` repository settings, add:

| Secret | Description |
|--------|-------------|
| `EUKIA_APP_ID` | The App ID integer (from App settings) |
| `EUKIA_APP_PRIVATE_KEY` | The content of the generated `.pem` file |

### 4. Initial Sync (First Time)

**Important**: The first sync requires `--init-history` to initialize the destination repos.

Via GitHub Actions:
1. Go to **Actions** → **Copybara Sync**
2. Click **Run workflow**
3. Select: `plugin: all`, `init_history: true`

Via CLI:
```bash
java -jar /tmp/copybara_deploy.jar migrate \
  infra/copybara/copy.bara.sky \
  push-kotlin \
  --init-history \
  --git-destination-url="https://github.com/albertocavalcante/gazelle-kotlin.git"
```

## Usage

### Manual sync (via GitHub Actions)

1. Go to **Actions** → **Copybara Sync**
2. Click **Run workflow**
3. Configure options:
   - `plugin`: Which plugin(s) to sync
   - `dry_run`: Validate without pushing
   - `init_history`: First-time sync
   - `last_rev`: Override starting point
   - `force`: Force push (dangerous)

### Local execution (via Bazel)

```bash
# Run copybara via Bazel
bazel run //tools/copybara -- migrate \
  $(pwd)/infra/copybara/copy.bara.sky \
  push-kotlin

# Or download and run directly
curl -fsSL -o /tmp/copybara_deploy.jar \
  "https://github.com/google/copybara/releases/download/v20251215/copybara_deploy.jar"

java -jar /tmp/copybara_deploy.jar migrate \
  infra/copybara/copy.bara.sky \
  validate-kotlin \
  --dry-run
```

### Automatic sync (on push)

The workflow automatically triggers on pushes to `main` that modify:
- `gazelle-kotlin/**`, `gazelle-python/**`, `gazelle-groovy/**`
- `internal/log/**`, `pkg/jvm/**`, `pkg/treesitter/**`
- `infra/copybara/**`

Only affected plugins are synced based on changed paths.

## Commit Message Tracking

Copybara adds a `Bazelle-RevId` trailer to track migrated commits:

```text
feat: Add new feature

Description here.

Bazelle-RevId: abc123def456
```

This prevents the same commit from being migrated twice.

## Troubleshooting

### Copybara Not Detecting Changes

Check that:
1. `origin_files` glob matches the changed files
2. Changes aren't excluded by `destination_files`
3. The commit hasn't already been migrated (check for `Bazelle-RevId` trailer)

### Authentication Issues

Ensure:
1. Eukia app has access to all 4 repositories
2. Secrets `EUKIA_APP_ID` and `EUKIA_APP_PRIVATE_KEY` are set correctly
3. The private key is the full PEM content including headers

### Empty Destination Repo

Use `--init-history` flag for first-time sync to create initial content.

## References

- [Copybara Repository](https://github.com/google/copybara)
- [Copybara Reference](https://github.com/google/copybara/blob/master/docs/reference.md)
- [Copybara Examples](https://github.com/google/copybara/blob/master/docs/examples.md)
