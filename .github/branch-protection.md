# Branch Protection / Required Checks Proposal

This repository is still in early bootstrap, so branch protection should stay strict enough to prevent obvious breakage without adding heavy process overhead.

## Recommended protected branch

- `main`

## Recommended required status checks

Require the following GitHub Actions job before merging into `main`:

- `Format, test, and build`

This maps to the job name in `.github/workflows/ci.yml`.

## Recommended branch protection settings for `main`

### Enable now

- **Require a pull request before merging**
- **Require status checks to pass before merging**
- **Require branches to be up to date before merging**
- **Dismiss stale pull request approvals when new commits are pushed**
- **Require conversation resolution before merging**
- **Block force pushes**
- **Block branch deletion**

### Team-size dependent

Use one of these depending on how the repo is being used:

#### Solo or mostly-solo development

- **Required approving reviews:** `0` or `1`

If the repo is effectively single-maintainer, `0` is acceptable as long as CI is required.
If review discipline matters, use `1`.

#### Multi-maintainer workflow

- **Required approving reviews:** `1`

## Recommended merge strategy

Preferred:

- **Squash merge** enabled

Optional:

- Merge commits disabled
- Rebase merge disabled

Reasoning: the repo is still evolving quickly, and squash merge keeps history tidy while preserving context in PRs.

## Suggested future required checks

Do **not** require these yet unless they exist and are stable:

- lint
- race test suite
- integration test suite
- coverage threshold checks
- security scanning gates

These can be added later when the implementation is less bootstrap-heavy.

## Why keep it minimal right now

The current codebase is still at a bootstrap stage.
A single reliable CI check is better than a pile of flaky or half-defined gates.
The goal right now is:

1. stop broken code from landing in `main`
2. keep contributor flow simple
3. avoid fake process maturity

## Suggested GitHub UI configuration summary

For branch `main`, configure:

- Require a pull request before merging: **ON**
- Require approvals: **1** (or **0** if intentionally solo)
- Dismiss stale approvals: **ON**
- Require status checks: **ON**
- Required checks:
  - `Format, test, and build`
- Require branches to be up to date: **ON**
- Require conversation resolution: **ON**
- Allow force pushes: **OFF**
- Allow deletions: **OFF**
