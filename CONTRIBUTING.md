# Contributing to GoAnomalyDetect

## Development Setup

1. Fork and clone the repository
2. Install dependencies:
   ```bash
   make deps
   ```
3. Install pre-commit hooks:
   ```bash
   pip install pre-commit
   pre-commit install
   ```

## Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Run `make lint` to check for issues

## Testing

- Write table-driven tests
- Aim for >90% coverage on new code
- Run `make test` before submitting PR

## Pull Request Process

1. Create a feature branch from `develop`
2. Write tests for new functionality
3. Update documentation if needed
4. Ensure CI passes
5. Request review

## Commit Messages

Use conventional commits:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `test:` tests
- `refactor:` code refactoring
- `perf:` performance improvement

## Reporting Issues

Include:
- Go version
- OS/platform
- Steps to reproduce
- Expected vs actual behavior
