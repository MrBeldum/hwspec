## What and why

<!-- What this changes, and the problem it solves. Link issues: Closes #123 -->

## How it was tested

<!-- Behaviour tests added or changed; manual checks (distros, hardware) -->

## Checklist

- [ ] Every commit follows [Conventional Commits](https://www.conventionalcommits.org/) (`type(scope): summary`), builds and passes tests on its own — PRs are **rebase-merged**, so each lands on `main`
- [ ] Tests cover the behaviour (not just the lines); `make test` and `make lint` pass
- [ ] File format changes are additive, or `schema_version` is bumped with an ADR
- [ ] Docs updated (README, ARCHITECTURE.md, or a new ADR in `docs/adr/`), using diagrams and tables where they help
- [ ] **Independent review** done (Claude Code: correctness, silent failures, security), every finding fixed, and the review with its resolution table posted as a PR comment
