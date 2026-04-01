#!/usr/bin/env node

/**
 * Purpose: Post-process generated OpenAPI TypeScript output so generated files meet repo hygiene requirements.
 * Responsibilities: Discover generated TypeScript files, remove generator TODO lines, prepend the required purpose header, and expose a CLI for `make generate`.
 * Scope: Generated OpenAPI client output only; this script does not modify hand-maintained source files.
 * Usage: Run `node scripts/strip_openapi_todos.mjs --path web/src/api` after OpenAPI generation.
 * Invariants/Assumptions: Target files are generated from `api/openapi.yaml`, generated files should not be edited manually, and re-running the script must be idempotent.
 */

import fs from "node:fs";
import path from "node:path";

const GENERATED_HEADER = `/**
 * Purpose: Provide generated OpenAPI client code for the Spartan Scraper web app.
 * Responsibilities: Expose generated API clients, request helpers, and schema types derived from \`api/openapi.yaml\`.
 * Scope: Generated code only; edit the OpenAPI contract or generator pipeline instead of this file.
 * Usage: Import generated APIs from adjacent web modules and regenerate with \`make generate\` after contract changes.
 * Invariants/Assumptions: This file is machine-generated, should stay in sync with the current contract, and may be overwritten at any time.
 */\n\n`;

function usage() {
  console.log(`
Post-process generated OpenAPI TypeScript output.

Usage:
  node scripts/strip_openapi_todos.mjs --path <dir>

Options:
  --path <dir>   Generated OpenAPI output directory to rewrite
  -h, --help     Show this help message and exit

Examples:
  node scripts/strip_openapi_todos.mjs --path web/src/api
  pnpm exec openapi-ts -i api/openapi.yaml -o web/src/api && node scripts/strip_openapi_todos.mjs --path web/src/api

Exit codes:
  0  Success
  1  Unexpected processing failure
  2  Invalid arguments or missing path
`);
}

function showError(message) {
  console.error(message);
}

function findTsFiles(dir, files = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      findTsFiles(fullPath, files);
    } else if (entry.isFile() && (entry.name.endsWith(".ts") || entry.name.endsWith(".mts"))) {
      files.push(fullPath);
    }
  }

  return files;
}

function stripTodos(content) {
  return content
    .split(/\r?\n/)
    .filter((line) => !line.match(/^.*TODO:.*$/))
    .join("\n");
}

function ensureGeneratedHeader(content) {
  if (content.startsWith(GENERATED_HEADER)) {
    return content;
  }
  return `${GENERATED_HEADER}${content.replace(/^\s*/, "")}`;
}

function normalizeGeneratedFile(content) {
  return ensureGeneratedHeader(stripTodos(content));
}

function stripTodosFromFile(filePath) {
  const content = fs.readFileSync(filePath, "utf8");
  const nextContent = normalizeGeneratedFile(content);

  if (nextContent === content) {
    return false;
  }

  fs.writeFileSync(filePath, nextContent, "utf8");
  return true;
}

function main() {
  const args = process.argv.slice(2);
  let targetPath = "";

  for (let index = 0; index < args.length; index++) {
    if (args[index] === "--help" || args[index] === "-h") {
      usage();
      process.exit(0);
    }
    if (args[index] === "--path") {
      if (index + 1 >= args.length) {
        showError("Error: --path requires an argument");
        usage();
        process.exit(2);
      }
      targetPath = args[index + 1];
      index++;
      continue;
    }
    showError(`Error: Unknown argument: ${args[index]}`);
    usage();
    process.exit(2);
  }

  if (!targetPath) {
    showError("Error: Missing --path");
    usage();
    process.exit(2);
  }

  if (!fs.existsSync(targetPath)) {
    showError(`Error: Path not found: ${targetPath}`);
    process.exit(2);
  }

  if (!fs.statSync(targetPath).isDirectory()) {
    showError(`Error: Path is not a directory: ${targetPath}`);
    process.exit(2);
  }

  try {
    for (const file of findTsFiles(targetPath)) {
      stripTodosFromFile(file);
    }
    process.exit(0);
  } catch (error) {
    showError(`Error: ${error.message}`);
    process.exit(1);
  }
}

export {
  GENERATED_HEADER,
  ensureGeneratedHeader,
  findTsFiles,
  main,
  normalizeGeneratedFile,
  showError,
  stripTodos,
  stripTodosFromFile,
  usage,
};

if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}
