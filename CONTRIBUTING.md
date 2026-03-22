# Contributing

Thanks for contributing to Papyrus.

## Before You Start

- Read [SPEC.md](SPEC.md) for the supported document and CSS model.
- Check open issues and pull requests before starting overlapping work.
- For security issues, do not open a public issue. Follow [SECURITY.md](SECURITY.md).

## Development Setup

Papyrus is a pure Go project.

Requirements:

- Go 1.22 or newer

Useful commands:

```bash
go test ./...
go test -race ./...
go vet ./...
```

If you intentionally change document layout snapshots, regenerate them with:

```bash
UPDATE_GOLDEN=1 go test ./pkg/document/...
```

## Development Guidelines

- Keep changes focused and easy to review.
- Add or update tests for behavior changes.
- Update documentation when public behavior changes.
- Preserve the project's strict subset approach. Unsupported markup and CSS
  should fail clearly, not degrade silently.
- Avoid introducing external runtime dependencies, CGO, or shell-outs.

## Pull Requests

Please make sure your pull request:

- Explains the problem and the chosen approach
- Links related issues when applicable
- Includes tests or a clear reason no tests were added
- Notes any breaking behavior changes
- Keeps generated or local-only assets out of the diff unless they are
  intentionally part of the repository

## Style

- Run `gofmt` on changed Go files
- Keep error messages concrete and contextual
- Prefer small, composable changes over broad refactors

## Reporting Bugs

When opening a bug report, include:

- What you expected to happen
- What actually happened
- A minimal XML or Go reproduction when possible
- Your Go version and operating system

## License

By contributing to this repository, you agree that your contributions will be
licensed under the [MIT License](LICENSE).
