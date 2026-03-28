# Contributing to StrataHQ Backend

Thank you for contributing! This guide will help you get started.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch: `git checkout -b feat/my-feature`
4. Follow the [Quick Start](README.md#quick-start) to set up your environment

## Development Workflow

### Branch Naming

- `feat/description` — new features
- `fix/description` — bug fixes
- `docs/description` — documentation
- `refactor/description` — code refactoring
- `test/description` — test additions or fixes

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(auth): add password reset endpoint
fix(levy): correct payment amount calculation
docs(readme): add deployment instructions
test(scheme): add integration tests for CRUD
refactor(middleware): simplify rate limit logic
```

### Code Style

- Run `make fmt` before committing
- Run `make lint` to check for issues
- CI enforces `golangci-lint` — your PR won't merge with lint errors

### Testing

- **Unit tests** go next to the code: `internal/mydomain/service_test.go`
- **Integration tests** go in `tests/integration/`
- Use table-driven tests:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"valid input", "hello", "HELLO"},
        {"empty input", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

- Run unit tests: `make test`
- Run integration tests: `make docker-up && make test-integration`
- All tests must pass before submitting a PR

### Adding a New Domain

See the [Adding a New Domain](README.md#adding-a-new-domain) section in the README.

## Pull Request Process

1. Ensure all tests pass (`make test-all`)
2. Ensure lint passes (`make lint`)
3. Update documentation if needed
4. Fill out the PR template
5. Request a review

### PR Template

```markdown
## What

Brief description of what this PR does.

## Why

Why is this change needed?

## How to Test

Steps to verify the change:
1. ...
2. ...
3. ...

## Checklist

- [ ] Tests added/updated
- [ ] Lint passes (`make lint`)
- [ ] Documentation updated (if applicable)
```

## Questions?

Open an issue or reach out to the maintainers.
