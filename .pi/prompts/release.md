---
description: Create a new release - commit changes, tag version, and push
---

Create a new release for this Go CLI project.

## Steps:

1. **Check current state**
   - Run `git status` to see uncommitted changes
   - Run `git tag -l | sort -V | tail -5` to see latest tags
   - Run `git log <latest-tag>..HEAD --oneline` to see commits since last tag

2. **Commit any pending changes** (if uncommitted files exist)
   - Review diffs with `git diff`
   - Stage all changes: `git add -A`
   - Create commit with Conventional Commits format: `git commit -m "<type>(<scope>): <summary>"`

3. **Determine version bump** based on changes:
   - **Major (X.0.0)**: Breaking changes
   - **Minor (0.X.0)**: New features, additions (use this for new commands/features)
   - **Patch (0.0.X)**: Bug fixes only
   - Ask user if unclear: "What version should I tag? Current: vX.Y.Z"

4. **Create and push tag**
   - `git tag v<VERSION>`
   - `git push origin v<VERSION>`

5. **Verify**
   - Confirm tag exists: `git tag -l v<VERSION>`
   - Show tag info: `git log v<VERSION> --oneline -1`

## Example releases for this project:
- `v0.1.0` - Initial release
- `v0.2.0` - Added Forgejo Actions commands (actions list, view, logs)

Always use lightweight tags for this project unless user requests annotated tags.
