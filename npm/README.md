# go-ripgrep (NPM Installer)

Pure Go implementation of [ripgrep](https://github.com/BurntSushi/ripgrep) (`rg`) with handwritten AVX2 SIMD optimizations, zero-allocation fast-paths, and zero CGO dependencies.

## Installation

Install globally via npm:

```bash
npm install -g go-ripgrep
```

Or run via `npx`:

```bash
npx go-ripgrep --help
```

## Features

- **No CGO Required**: Pure Go static binary, easy single-binary deployment.
- **AVX2 Acceleration**: Handwritten Go Assembly vector instructions for fast ASCII searches (up to 15x speedup).
- **Match Replacement**: Full regex and literal matching replacement (`-r` / `--replace`).
- **File Type Filtering**: Pre-configured file extension rules (e.g. `-t go`, `-T rust`).
- **Ignore climbed trees**: Correct climbing and parsing of nested `.gitignore`, `.ignore`, `.rgignore` rules.
- **Decompression Search**: Search inside `.zip`, `.gz`, and `.bz2` files directly.

## Usage

```bash
rg [OPTIONS] PATTERN [PATH...]
```

For more options, run:

```bash
rg --help
```
