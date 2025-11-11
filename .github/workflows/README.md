# GitHub Workflows Documentation

This directory contains GitHub Actions workflows for automating various tasks in the Vigilon project.

## Workflows

### 1. Release Workflow (`release.yml`)

**Triggers:** On version tags (e.g., `v1.0.0`, `v1.1.0`)

**What it does:**
- Automatically builds binaries for all platforms when you create a new version tag
- Creates a GitHub Release with the binaries attached
- Updates `web/static/bin/` with the latest agent binaries
- Generates SHA256 checksums for verification

**Usage:**
```bash
git tag v1.1.0
git push origin v1.1.0
```

**Output:**
- GitHub Release with 4 binaries + checksums
- Updated binaries in `web/static/bin/` for install scripts
- Automatic release notes from `RELEASE_NOTES.md`

### 2. CI Workflow (`ci.yml`)

**Triggers:** On every push to `main` or `develop` branches, and on pull requests

**What it does:**
- Runs tests with race detection
- Checks code formatting (`gofmt`)
- Runs static analysis (`go vet`, `staticcheck`)
- Verifies dependencies
- Builds binaries for all platforms to ensure compilation succeeds
- Uploads test coverage to Codecov
- Stores build artifacts for 7 days

**Checks:**
- ✅ Code formatting
- ✅ Linting (staticcheck)
- ✅ Tests with race detection
- ✅ Build verification for all platforms

### 3. Docker Workflow (`docker.yml`)

**Triggers:** 
- On version tags (e.g., `v1.0.0`)
- On push to `main` branch
- Manual trigger via GitHub UI

**What it does:**
- Builds multi-platform Docker images (AMD64, ARM64)
- Pushes to Docker Hub
- Creates version tags (`latest`, `1.0.0`, `1.0`, `1`)
- Updates Docker Hub descriptions

**Docker Hub Images:**
- `harungecit/vigilon-server:latest`
- `harungecit/vigilon-server:v1.0.0`
- `harungecit/vigilon-agent:latest`
- `harungecit/vigilon-agent:v1.0.0`

**Requirements:**
- `DOCKER_USERNAME` secret in GitHub repository settings
- `DOCKER_PASSWORD` secret (Docker Hub access token)

## Setup Instructions

### For Release Workflow
No setup required! Uses `GITHUB_TOKEN` which is automatically available.

### For Docker Workflow
1. Go to GitHub repository → Settings → Secrets and variables → Actions
2. Add two secrets:
   - `DOCKER_USERNAME`: Your Docker Hub username (e.g., `harungecit`)
   - `DOCKER_PASSWORD`: Your Docker Hub access token (not password!)
     - Create token at: https://hub.docker.com/settings/security
     - Permissions needed: Read, Write, Delete

### For CI Workflow (Optional)
To enable Codecov coverage reports:
1. Sign up at https://codecov.io with your GitHub account
2. Enable the Vigilon repository
3. No secrets needed - uses `GITHUB_TOKEN`

## Workflow Examples

### Creating a New Release

1. Update version in code if needed
2. Update `RELEASE_NOTES.md`
3. Commit changes
4. Create and push tag:
```bash
git tag v1.1.0 -m "Release version 1.1.0"
git push origin v1.1.0
```

5. Wait ~5 minutes for workflows to complete
6. Check GitHub Releases page for the new release
7. Check Docker Hub for new images

### Testing Before Release

Push to `main` branch to trigger CI workflow:
```bash
git push origin main
```

This will:
- Run all tests
- Build all binaries
- Report any issues before you create a release

### Manual Docker Build

Go to GitHub → Actions → Docker Build and Push → Run workflow

## Maintenance

### Updating Go Version
Edit all three workflow files and change:
```yaml
go-version: '1.23'
```

### Adding New Platforms
Edit `release.yml` and add to the build steps:
```yaml
- name: Build Agent Binary (macOS ARM64)
  run: |
    GOOS=darwin GOARCH=arm64 go build \
      -ldflags="-s -w" \
      -o vigilon-agent-darwin-arm64 \
      cmd/agent/main.go
```

### Disabling Workflows
Temporarily disable by renaming file:
```bash
mv .github/workflows/docker.yml .github/workflows/docker.yml.disabled
```

## Troubleshooting

### Release workflow fails with "permission denied"
- Check that `permissions: contents: write` is set in workflow
- Verify `GITHUB_TOKEN` has write permissions (should be automatic)

### Docker workflow fails with "authentication required"
- Verify `DOCKER_USERNAME` and `DOCKER_PASSWORD` secrets are set
- Ensure Docker Hub token hasn't expired
- Check token permissions (Read, Write, Delete)

### CI workflow fails on tests
- Check test output in GitHub Actions logs
- Run tests locally: `go test -v ./...`
- Run race detector: `go test -race ./...`

### Binaries not updated in web/static/bin/
- Check that release workflow completed successfully
- Verify git push permissions
- Check workflow logs for commit step

## Benefits

✅ **Automated Releases**: No manual binary building
✅ **Consistent Builds**: Same environment every time
✅ **Multi-Platform**: Linux, Windows, ARM64 automatically
✅ **Quality Assurance**: Tests run before merge
✅ **Docker Images**: Automatically published to Docker Hub
✅ **Version Management**: Automatic tagging and checksums
✅ **Time Saving**: 5 minutes instead of 30 minutes manual work

## Workflow Status Badges

Add these to your README.md:

```markdown
[![Release](https://github.com/harungecit/vigilon/actions/workflows/release.yml/badge.svg)](https://github.com/harungecit/vigilon/actions/workflows/release.yml)
[![CI](https://github.com/harungecit/vigilon/actions/workflows/ci.yml/badge.svg)](https://github.com/harungecit/vigilon/actions/workflows/ci.yml)
[![Docker](https://github.com/harungecit/vigilon/actions/workflows/docker.yml/badge.svg)](https://github.com/harungecit/vigilon/actions/workflows/docker.yml)
```
