# Documentation Publishing Architecture

Papyrus publishes documentation as a static MkDocs site on GitHub Pages.

That statement hides an important engineering distinction: "the docs built
successfully" and "the docs are being served successfully" are related, but
they are not the same thing.

This page explains the two GitHub Pages deployment architectures, why the
current repository uses one of them, where the operational edges are, and what
the cleaner long-term target looks like.

## The Core Model

A documentation deployment system has three separate responsibilities:

1. Build: convert Markdown, theme assets, and navigation into static HTML, CSS,
   and JavaScript.
2. Publish: move that build output into a place GitHub Pages understands.
3. Serve: have GitHub Pages actually expose that output at the public URL.

Most confusion comes from collapsing those into one mental step. They are not
one step.

In Papyrus:

- MkDocs handles the build phase.
- GitHub Actions handles the automation phase.
- GitHub Pages handles the serving phase.

That separation is healthy from a systems-design standpoint, but it also means
you can succeed in one layer and still fail in another.

## Architecture 1: Branch-Based Publishing

This is the model the repository currently uses.

```text
push to master
  -> GitHub Actions workflow runs
    -> MkDocs builds the static site
    -> mkdocs gh-deploy commits generated files to gh-pages
      -> GitHub Pages must be enabled
      -> GitHub Pages must be configured to publish from gh-pages / root
        -> site becomes live
```

### Why teams use it

- It is the most direct integration path for MkDocs.
- It is simple to reason about when bootstrapping a project.
- The generated site is inspectable in a normal Git branch.
- Rollbacks are conceptually easy because the published output is versioned.

### Strengths

- Low setup friction
- Tool-native for MkDocs
- Easy to inspect the exact deployed HTML output
- Good for small projects and early-stage repos

### Weaknesses

- Workflow success does not imply public-site success
- The deployment contract is split across CI and repo settings
- Generated site output lives in Git history
- Misconfiguration is easy to miss until somebody checks the public URL

### Key operational nuance

`mkdocs gh-deploy` is a branch publication mechanism, not a full Pages
provisioning mechanism.

Said differently: it can update `gh-pages`, but it cannot guarantee that GitHub
Pages is enabled for the repository or that the repository is configured to
serve from that branch.

That is why this architecture can produce the following outcome:

```text
workflow: green
gh-pages branch: updated
public URL: 404
```

That is not contradictory. It is the natural result of the deployment and
serving control planes being separate.

## Architecture 2: GitHub Pages Actions Publishing

This is the newer, more integrated GitHub-native model.

```text
push to master
  -> GitHub Actions workflow runs
    -> MkDocs builds the static site
    -> build output is uploaded as a Pages artifact
    -> deploy-pages job publishes the artifact
      -> GitHub Pages serves the deployed artifact
```

### Why teams use it

- It aligns with GitHub Pages' current deployment model
- It reduces hidden coupling to branch settings
- It avoids committing generated site output to a long-lived branch
- It makes the deployment path more explicit in CI

### Strengths

- Cleaner separation between source code and generated output
- Fewer branch-management concerns
- Better audit trail at the workflow/job level
- Stronger coupling between successful deploy job and actual publication

### Weaknesses

- Slightly more workflow complexity
- The deployed artifact is less convenient to inspect than a plain branch
- It is marginally less familiar to teams that have historically used
  `gh-pages`

## Why Papyrus Currently Uses Branch-Based Publishing

The current choice is pragmatic, not ideological.

Papyrus is still in a phase where the product, examples, docs, and branding are
moving quickly. In that phase, the branch-based model has three practical
advantages:

1. It gets a static site online with minimal ceremony.
2. It keeps the generated output easy to inspect when something looks wrong.
3. It matches the MkDocs mental model directly, which reduces bootstrap cost.

For an early-stage open source project, simplicity has real value. The best
architecture is not always the most "modern" one. It is often the one that
minimizes unnecessary moving parts while the problem space is still settling.

That said, this simplicity comes with a real operational tradeoff: publication
is only half of the deployment story. Serving still depends on correct GitHub
Pages configuration.

## Why The Pipeline Can Pass While The Site Is Still Down

This is the exact failure mode that surfaced in the repository.

The workflow succeeded because CI did its job:

- source was checked out
- MkDocs built the site
- `gh-pages` was updated

The public URL still returned `404` because GitHub Pages was not serving the
published branch.

The precise systems lesson is:

> Build correctness and serving correctness are different contracts.

Or more concretely:

- CI answers: "Did we produce and publish static output?"
- Pages answers: "Am I configured to expose that output publicly?"

If the second system is not enabled or misconfigured, the first system can be
perfect and users will still see nothing.

## Distinguished Engineer View

At a systems level, the decision is not really about MkDocs versus GitHub
Pages. It is about where you want the deployment truth to live.

### In the branch-based model

The truth is split:

- Git workflow state says what was published
- GitHub Pages settings say whether that state is actually served

That is operationally acceptable, but it is a looser contract.

### In the Actions-based model

The truth is more centralized:

- the workflow builds
- the workflow deploys
- GitHub Pages consumes that deployment path directly

That is a tighter operational contract and usually the better long-term choice
for mature systems.

## Recommendation

If the goal is fastest bootstrap and easiest human inspection, the current
branch-based solution is reasonable.

If the goal is stronger deployment guarantees and fewer hidden dependencies on
repository settings, the recommended long-term direction is the official GitHub
Pages Actions deployment flow.

That recommendation is not because the current model is "wrong." It is because
the Actions-based model reduces the number of ways a deployment can look
successful while still being unavailable to users.

## Practical Checklist

For the current branch-based setup to work correctly:

1. GitHub Pages must be enabled for the repository.
2. The publishing source must be `gh-pages`.
3. The folder must be `/(root)`.
4. The public URL should be checked after each first-time setup or source
   change.

If any of those are false, a successful docs workflow is not enough.
