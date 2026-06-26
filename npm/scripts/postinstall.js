#!/usr/bin/env node

// Skip postinstall output in CI or when suppressed
if (process.env.CI || process.env.npm_config_yes || process.env.GO_RIPGREP_SKIP_POSTINSTALL) {
  process.exit(0);
}

const RESET  = '\x1b[0m';
const BOLD   = '\x1b[1m';
const DIM    = '\x1b[2m';
const CYAN   = '\x1b[36m';
const BRIGHT_CYAN = '\x1b[96m';
const WHITE  = '\x1b[97m';

const logo = [
  "   ____ _____      _                       ",
  "  / __ `/ __ \\____(_)___  _________ ___  ",
  " / /_/ / /_/ /  __` / __ \\/ ___/  _`_  \\",
  " \\__, /\\____/ /_/ / /_/ / /  /  / / / /",
  "/____/      \\__, / .___/_/  /_/ /_/ /_/ ",
  "           /____/_/                    "
].join('\n');

function pkgVersion() {
  try {
    return require('../package.json').version;
  } catch {
    return '';
  }
}

const ver = pkgVersion();
const verStr = ver ? ` ${DIM}v${ver}${RESET}` : '';

console.log();
console.log(`${BRIGHT_CYAN}${BOLD}${logo}${RESET}${verStr}`);
console.log();
console.log(`  ${BOLD}${WHITE}High-performance line-oriented search tool written in Go.${RESET}`);
console.log();
console.log(`  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}`);
console.log();
console.log(`  ${BOLD}Quick Start${RESET}`);
console.log();
console.log(`    rg "pattern"                   ${DIM}Search recursively${RESET}`);
console.log(`    rg -i "pattern" path/          ${DIM}Case-insensitive search${RESET}`);
console.log(`    rg --json "pattern"            ${DIM}JSON structured output${RESET}`);
console.log();
console.log(`  ${BOLD}Documentation & SDK Usage${RESET}`);
console.log();
console.log(`    Refer to GitHub Repository for standard Go package SDK imports.`);
console.log(`    ${CYAN}https://github.com/ripgrep/go-ripgrep${RESET}`);
console.log();
