Always use semantic (Conventional Commits) messages for git commits.

Preferred format:
type(scope): short summary

Examples:
fix(kotlin): handle inline block comments
test(kotlin): add parser regression cases
chore(repo): update tooling config

Copybara validation:
- Use bazel run //tools/copybara:copybara -- migrate /absolute/path/to/infra/copybara/copy.bara.sky <workflow> --dry-run
- Remove no-op transformations like core.move("path/", "path/") to avoid config errors
