# NPM Integration

This document describes how to use go-ripgrep via npm and how the npm packages are structured.

## Installation

### As a Global Tool

```bash
npm install -g go-ripgrep
```

After installation, the `rg` command is available globally:

```bash
rg --version
rg "pattern" ./src
```

### As a Project Dependency

```bash
npm install go-ripgrep
```

Use via npm scripts:

```json
{
  "scripts": {
    "search": "rg --json 'TODO' ./src"
  }
}
```

Or reference the binary directly:

```bash
./node_modules/.bin/rg "pattern"
```

## Package Structure

The npm distribution consists of a main package and platform-specific optional dependencies:

```
go-ripgrep (main package)
├── go-ripgrep-linux-x64      (optional)
├── go-ripgrep-linux-arm64    (optional)
├── go-ripgrep-linux-loong64  (optional)
├── go-ripgrep-linux-musl-x64 (optional)
├── go-ripgrep-darwin-x64     (optional)
├── go-ripgrep-darwin-arm64   (optional)
├── go-ripgrep-win32-x64      (optional)
└── go-ripgrep-win32-arm64    (optional)
```

### Main Package (`go-ripgrep`)

The main package:
- Contains the `bin/` directory with the platform-appropriate binary
- Includes a `postinstall` script to set up the binary
- Declares all platform packages as `optionalDependencies`

### Platform Packages

Each platform package:
- Contains the pre-built binary for that platform
- Is named following the convention: `go-ripgrep-{os}-{arch}`
- Is automatically installed by npm based on the current platform

## Supported Platforms

| Package Name | OS | Architecture |
|--------------|-----|-------------|
| `go-ripgrep-linux-x64` | Linux | x86_64 |
| `go-ripgrep-linux-arm64` | Linux | ARM64 |
| `go-ripgrep-linux-loong64` | Linux | LoongArch64 |
| `go-ripgrep-linux-musl-x64` | Linux (Alpine/musl) | x86_64 |
| `go-ripgrep-darwin-x64` | macOS | Intel |
| `go-ripgrep-darwin-arm64` | macOS | Apple Silicon |
| `go-ripgrep-win32-x64` | Windows | x86_64 |
| `go-ripgrep-win32-arm64` | Windows | ARM64 |

## Postinstall Script

The `postinstall` script (`scripts/postinstall.js`) handles:
1. Detecting the current platform
2. Copying the correct binary from the platform package
3. Setting executable permissions (on Unix systems)

## Building NPM Packages

### Prerequisites

- Node.js 14+
- npm
- Go (for building binaries)
- UPX (optional, for binary compression)

### Build Process

```bash
# 1. Sync version across all package.json files
make npm-version

# 2. Build binaries for all platforms
make build-all

# 3. Build platform-specific npm packages
make npm-packages

# 4. Pack everything into tarballs
make npm-pack
```

### Version Management

The version is defined in the main `npm/package.json` and synced to all platform packages:

```bash
# Current version
grep '"version"' npm/package.json

# Sync version
make npm-version
```

### Publishing

```bash
# Publish all packages (main + platform-specific)
make npm-publish-all
```

This will:
1. Sync the version
2. Build all platform binaries
3. Build npm packages
4. Publish each platform package
5. Publish the main package

## Programmatic Usage in Node.js

While go-ripgrep is primarily a CLI tool, you can use it programmatically in Node.js:

```javascript
const { execFile } = require('child_process');
const path = require('path');

// Get the path to the rg binary
const rgBin = path.join(__dirname, 'node_modules', '.bin', 'rg');

// Execute a search
execFile(rgBin, ['--json', 'TODO', './src'], (error, stdout, stderr) => {
  if (error && error.code === 1) {
    // Exit code 1 means no matches found
    console.log('No matches found');
    return;
  }

  if (error) {
    console.error('Error:', stderr);
    return;
  }

  // Parse NDJSON output
  const results = stdout.trim().split('\n').map(line => JSON.parse(line));
  results.forEach(result => {
    if (result.type === 'match') {
      console.log(`${result.data.path.text}:${result.data.line_number}: ${result.data.lines.text}`);
    }
  });
});
```

### Using with Child Process

```javascript
const { spawn } = require('child_process');

function search(pattern, directory) {
  return new Promise((resolve, reject) => {
    const rg = spawn('rg', ['--json', pattern, directory]);
    const results = [];

    rg.stdout.on('data', (data) => {
      const lines = data.toString().trim().split('\n');
      lines.forEach(line => {
        if (line) {
          try {
            results.push(JSON.parse(line));
          } catch (e) {
            // ignore parse errors
          }
        }
      });
    });

    rg.on('close', (code) => {
      if (code === 1) {
        resolve([]); // no matches
      } else if (code === 0) {
        resolve(results);
      } else {
        reject(new Error(`rg exited with code ${code}`));
      }
    });
  });
}

// Usage
search('TODO', './src').then(results => {
  const matches = results.filter(r => r.type === 'match');
  console.log(`Found ${matches.length} matches`);
});
```

## Troubleshooting

### Binary Not Found

If the `rg` command is not found after installation:

```bash
# Check if binary exists
ls -la node_modules/.bin/rg

# On Windows
dir node_modules\.bin\rg.cmd
```

### Permission Denied

On Unix systems, ensure the binary is executable:

```bash
chmod +x node_modules/.bin/rg
```

### Wrong Platform

If npm installed the wrong platform package:

```bash
# Check installed packages
npm ls go-ripgrep-*

# Force reinstall
npm reinstall go-ripgrep
```

### musl/Alpine Linux

On Alpine Linux or other musl-based systems, the `go-ripgrep-linux-musl-x64` package is used automatically. If it's not installed:

```bash
npm install go-ripgrep-linux-musl-x64
```
