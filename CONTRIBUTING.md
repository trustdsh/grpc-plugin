# Contributing to gRPC Plugin System

First off, thank you for considering contributing to the gRPC Plugin System! It's people like you that make this project better for everyone.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* Use a clear and descriptive title
* Describe the exact steps which reproduce the problem
* Provide specific examples to demonstrate the steps
* Describe the behavior you observed after following the steps
* Explain which behavior you expected to see instead and why
* Include logs if relevant
* Include your Go environment (`go version`, OS, etc.)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* A clear and descriptive title
* A detailed description of the proposed functionality
* Explain why this enhancement would be useful
* List any alternative solutions or features you've considered
* Include any relevant examples

### Pull Requests

#### Development Setup

1. Fork the repo
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/grpc-plugin.git
   cd grpc-plugin
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/trustdsh/grpc-plugin.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```

#### Development Process

1. Create a new branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. Make your changes
3. Run tests:
   ```bash
   go test ./...
   ```
4. Run linters:
   ```bash
   golangci-lint run
   ```
5. Commit your changes:
   ```bash
   git commit -m "feat: add your feature description"
   ```
   Please follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

#### Pull Request Guidelines

* Update the README.md with details of changes to the interface, if applicable
* Update the CHANGELOG.md following the Keep a Changelog format
* The PR should work for Go 1.21 and later
* Include tests for new functionality
* Follow the existing code style
* Keep PRs focused - one feature/fix per PR

## Coding Style

* Follow standard Go conventions
* Use `gofmt` to format your code
* Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
* Write meaningful commit messages following Conventional Commits
* Document exported functions and types

### Code Structure

```
.
├── cmd/                    # Command line tools
├── examples/               # Example implementations
├── internal/              # Internal packages
│   └── transport/         # gRPC transport layer
├── pkg/                   # Public API packages
│   ├── config/           # Configuration
│── plugin/               # Plugin implementation
│── runner/               # Plugin runner implementation
├── test/                 # Integration tests
└── tools/                # Development tools
```

### Testing

* Write unit tests for new functionality
* Include integration tests for complex features
* Use table-driven tests where appropriate
* Aim for high test coverage (>80%)
* Use meaningful test names and descriptions

### Documentation

* Document all exported types and functions
* Include examples in documentation
* Keep documentation up to date with code changes
* Use clear and concise language

## Release Process

1. Update CHANGELOG.md
2. Update version numbers
3. Create a new release branch
4. Create and push a new tag
5. GitHub Actions will automatically build and publish

## Additional Notes

### Issue Labels

* `bug`: Something isn't working
* `enhancement`: New feature or request
* `documentation`: Documentation improvements
* `good first issue`: Good for newcomers
* `help wanted`: Extra attention is needed

### Communication

* Use GitHub Issues for bug reports and feature requests
* Use GitHub Discussions for questions and community interaction
* Join our [Discord](https://discord.gg/uddjvRC2WU) for real-time discussion

## Questions?

Don't hesitate to ask questions if something is unclear. You can:

* Open an issue
* Start a GitHub Discussion
* Contact the maintainers directly

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License. 