# Contributing to Morpheus

Thank you for your interest in contributing to Morpheus! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and constructive in all interactions.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/yourusername/morpheus/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Morpheus version (`morpheus version`)
   - Environment details (OS, Go version, etc.)
   - Relevant logs or error messages

### Suggesting Features

1. Check [Discussions](https://github.com/yourusername/morpheus/discussions) for similar ideas
2. Open a new discussion to propose your feature
3. Explain:
   - The problem it solves
   - Proposed implementation
   - Alternative approaches considered
   - Impact on existing functionality

### Pull Requests

#### Before You Start

1. Fork the repository
2. Create a feature branch from `main`
3. Discuss major changes in an issue first

#### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/morpheus.git
cd morpheus

# Add upstream remote
git remote add upstream https://github.com/yourusername/morpheus.git

# Install dependencies
make deps

# Build
make build
```

#### Making Changes

1. **Write Clean Code**
   - Follow Go best practices
   - Use meaningful variable names
   - Add comments for complex logic
   - Keep functions focused and small

2. **Format Your Code**
   ```bash
   make fmt
   ```

3. **Run Linters**
   ```bash
   make lint
   ```

4. **Add Tests**
   - Write tests for new functionality
   - Ensure existing tests pass
   ```bash
   make test
   ```

5. **Update Documentation**
   - Update README.md if needed
   - Add code comments
   - Update SETUP.md for setup changes

#### Commit Messages

Write clear, descriptive commit messages:

```
Add support for AWS provider

- Implement AWS provider interface
- Add cloud-init templates for EC2
- Update configuration schema
- Add tests for AWS provisioning

Fixes #123
```

Format:
- Use present tense ("Add feature" not "Added feature")
- Keep first line under 50 characters
- Add detailed description after blank line
- Reference issues/PRs at the end

#### Submitting Your PR

1. **Update Your Branch**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push to Your Fork**
   ```bash
   git push origin feature/your-feature
   ```

3. **Create Pull Request**
   - Go to GitHub and create a PR
   - Fill in the PR template
   - Link related issues
   - Request review

4. **PR Checklist**
   - [ ] Code builds without errors
   - [ ] Tests pass
   - [ ] Linters pass
   - [ ] Documentation updated
   - [ ] Commits are clean and descriptive
   - [ ] PR description explains changes

#### Review Process

- Maintainers will review your PR
- Address feedback promptly
- Be patient and respectful
- Make requested changes
- Once approved, a maintainer will merge

## Project Structure

```
morpheus/
├── cmd/                   # Command-line applications
│   └── morpheus/         # Main CLI
├── pkg/                   # Library packages
│   ├── config/           # Configuration management
│   ├── cloudinit/        # Cloud-init templates
│   ├── forest/           # Forest registry and provisioning
│   └── provider/         # Cloud provider implementations
├── docs/                  # Additional documentation
├── examples/             # Example configurations
└── tests/                # Integration tests
```

## Adding a New Cloud Provider

To add support for a new cloud provider:

1. **Create Provider Package**
   ```
   pkg/provider/newprovider/newprovider.go
   ```

2. **Implement Provider Interface**
   ```go
   type Provider interface {
       CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error)
       GetServer(ctx context.Context, serverID string) (*Server, error)
       DeleteServer(ctx context.Context, serverID string) error
       WaitForServer(ctx context.Context, serverID string, state ServerState) error
       ListServers(ctx context.Context, filters map[string]string) ([]*Server, error)
   }
   ```

3. **Add Cloud-Init Template**
   - Update `pkg/cloudinit/templates.go` if needed
   - Test bootstrap process

4. **Update Configuration**
   - Add provider-specific config options
   - Update config validation

5. **Update Documentation**
   - Add to README.md
   - Update SETUP.md
   - Add examples

6. **Add Tests**
   - Unit tests for provider
   - Integration tests (if possible)

## Coding Standards

### Go Style

Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines:

- Use `gofmt` for formatting
- Use `golint` and `go vet`
- Handle errors explicitly
- Use meaningful names
- Write documentation comments

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to create server: %w", err)
}

// Bad
if err != nil {
    log.Fatal(err)
}
```

### Logging

```go
// Use fmt.Printf for user-facing messages
fmt.Printf("✓ Server %s created\n", serverID)

// Consider using a structured logger for internal logging
```

### Configuration

- Use YAML for configuration files
- Support environment variables
- Validate config on load
- Provide sensible defaults

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/provider/hetzner/...

# Run with coverage
go test -cover ./...
```

### Integration Tests

Create integration tests in `tests/` directory:

```go
// tests/integration_test.go
// +build integration

func TestHetznerProvisioning(t *testing.T) {
    // ...
}
```

Run with:
```bash
go test -tags=integration ./tests/...
```

## Documentation

### Code Comments

```go
// CreateServer provisions a new server with the specified configuration.
// It returns the created server details or an error if provisioning fails.
//
// The function will:
//   - Validate the server configuration
//   - Create the server via the provider API
//   - Wait for the server to become ready
//   - Apply cloud-init configuration
//
// Example:
//   server, err := provider.CreateServer(ctx, CreateServerRequest{
//       Name: "my-server",
//       ServerType: "cpx31",
//   })
func CreateServer(ctx context.Context, req CreateServerRequest) (*Server, error) {
    // ...
}
```

### README Updates

Update these sections when making relevant changes:
- Features
- Installation
- Configuration
- Usage examples
- Troubleshooting

## Release Process

Maintainers will handle releases:

1. Update version in `cmd/morpheus/main.go`
2. Update CHANGELOG.md
3. Create git tag: `git tag v1.x.x`
4. Push tag: `git push origin v1.x.x`
5. Create GitHub release
6. Build and upload binaries

## Getting Help

- **Questions**: Use [GitHub Discussions](https://github.com/yourusername/morpheus/discussions)
- **Bugs**: Open an [Issue](https://github.com/yourusername/morpheus/issues)
- **Chat**: Join our community chat (if available)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to Morpheus! 🌲
