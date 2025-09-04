# Contributing to LilWiki

Thank you for your interest in contributing! LilWiki is a lightweight, open-source personal wiki. We welcome bug reports, feature requests, documentation improvements, and code contributions.

## How to Contribute

1. **Fork the repository** and create your branch from `main`.
2. **Make your changes** (see coding guidelines below).
3. **Test your changes** locally.
4. **Submit a pull request** with a clear description of your changes.

## Coding Guidelines

- **Go Backend**
  - Use idiomatic Go style (`gofmt` before committing).
  - Add doc comments for exported functions and types.
  - Use constants for special values (e.g., page names).
  - Standardize error responses as JSON.
  - Validate user input.
  - Modularize code for maintainability.

- **Frontend (JS, HTML, CSS)**
  - Use ES6+ syntax and modularize code.
  - Avoid inline event handlers; use event delegation.
  - Improve accessibility (ARIA, keyboard navigation).
  - Add comments for complex logic.
  - Test UI changes in multiple browsers.

## Reporting Issues

- Use [GitHub Issues](https://github.com/your-repo/issues) to report bugs or request features.
- Include steps to reproduce, expected behavior, and screenshots if helpful.

## Pull Request Checklist

- [ ] Code is formatted and linted
- [ ] All tests pass (if applicable)
- [ ] Documentation is updated
- [ ] PR description clearly explains the change

## License

By contributing, you agree your code will be released under the MIT License.

---

Thanks for helping make LilWiki better!
