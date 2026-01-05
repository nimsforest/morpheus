# Claude Code Guidelines

## Git Workflow

- Never commit directly to main
- Always create a branch first, then make changes
- Use worktrees for parallel agent work

## Branch Naming

- `feature/<name>` - new features
- `fix/<name>` - bug fixes
- `chore/<name>` - maintenance

## For New Features

1. Enter plan mode
2. Create plan at `todo/<feature-name>.md` (see template below)
3. Mark status `ready` when planning complete
4. Orchestrator creates worktrees and spawns agents

## Plan Template

```markdown
# Feature: <name>

## Status: planning

## Summary
<What this feature does>

## Tasks
- [ ] Task 1 - description (~files)
- [ ] Task 2 - description (~files)

## Parallelization
Group A: Task 1, Task 2 (no conflicts)
Group B: Task 3 (depends on A)

## Files
- pkg/foo/bar.go
- pkg/baz/qux.go
```

Statuses: `planning` | `ready` | `in-progress` | `done`

## Worktree Setup (Orchestrator)

```bash
FEATURE="feature-name"
git checkout main && git pull
git checkout -b feature/$FEATURE
git worktree add ../morpheus-$FEATURE-a -b feature/$FEATURE-a
git worktree add ../morpheus-$FEATURE-b -b feature/$FEATURE-b
```

## Commit Messages

```
Type: Short description

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <co-author>
```

Types: `Feature`, `Fix`, `Chore`, `Refactor`, `Docs`, `Test`
