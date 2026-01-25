#!/usr/bin/env bash
# Workspace status script for Bazel stamping.
# Provides version information for go_binary x_defs.
#
# Usage: Configure in .bazelrc:
#   build --workspace_status_command=tools/scripts/workspace_status.sh

set -euo pipefail

# Stable status (cached, only rebuilds when these change)
echo "STABLE_GIT_COMMIT $(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
echo "STABLE_GIT_BRANCH $(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')"

# Check if we're on a tag
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
if [[ -n "$GIT_TAG" ]]; then
    echo "STABLE_VERSION ${GIT_TAG}"
else
    # Use short commit hash as version for non-tagged builds
    SHORT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'dev')
    echo "STABLE_VERSION dev-${SHORT_COMMIT}"
fi

# Volatile status (always changes, doesn't trigger rebuilds)
echo "BUILD_TIMESTAMP $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "BUILD_USER ${USER:-unknown}"
echo "BUILD_HOST $(hostname -s 2>/dev/null || echo 'unknown')"
