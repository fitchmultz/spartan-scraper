#!/usr/bin/env node

/**
 * Purpose: Verify the OpenAPI post-processing script removes generator TODOs and prepends the required generated-file header.
 * Responsibilities: Exercise file discovery, TODO stripping, header normalization, and CLI help output for `strip_openapi_todos.mjs`.
 * Scope: Script-level regression coverage only; broader OpenAPI generation behavior stays outside this file.
 * Usage: Run `node scripts/strip_openapi_todos.test.mjs` or via `make ci`.
 * Invariants/Assumptions: Tests run in a temporary directory, generated files are TypeScript or MTS files, and the script stays idempotent.
 */

function usage() {
  console.log(`
Verify scripts/strip_openapi_todos.mjs.

Usage:
  node scripts/strip_openapi_todos.test.mjs [options]

Options:
  -h, --help   Show this help message and exit

Examples:
  node scripts/strip_openapi_todos.test.mjs
  node scripts/strip_openapi_todos.test.mjs --help

Exit codes:
  0  Success
  1  One or more tests failed
`);
}

import fs from "node:fs";
import path from "node:path";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const args = process.argv.slice(2);
if (args.includes("--help") || args.includes("-h")) {
  usage();
  process.exit(0);
}

let testsPassed = 0;
let testsFailed = 0;

function test(name, fn) {
  try {
    const result = fn();
    if (result instanceof Promise) {
      throw new Error("Async test returned a Promise; wrap it in runAsyncTest.");
    }
    testsPassed++;
    console.log(`✓ ${name}`);
  } catch (error) {
    testsFailed++;
    console.error(`✗ ${name}`);
    console.error(`  ${error.message}`);
  }
}

async function runAsyncTest(name, fn) {
  try {
    await fn();
    testsPassed++;
    console.log(`✓ ${name}`);
  } catch (error) {
    testsFailed++;
    console.error(`✗ ${name}`);
    console.error(`  ${error.message}`);
  }
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message || "Assertion failed");
  }
}

function assertEqual(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(message || `Expected ${expected}, got ${actual}`);
  }
}

async function runTests() {
  const tempDir = path.join(__dirname, ".test-tmp");
  fs.rmSync(tempDir, { recursive: true, force: true });
  fs.mkdirSync(tempDir, { recursive: true });

  const {
    GENERATED_HEADER,
    ensureGeneratedHeader,
    findTsFiles,
    normalizeGeneratedFile,
    stripTodosFromFile,
  } = await import("./strip_openapi_todos.mjs");

  test("Script removes TODO lines from files", () => {
    const testFile = path.join(tempDir, "todo.ts");
    fs.writeFileSync(
      testFile,
      `export interface User {\n  name: string;\n}\n// TODO: implement validation\nexport const userCount = 1;\n`,
      "utf8",
    );

    try {
      stripTodosFromFile(testFile);
      const result = fs.readFileSync(testFile, "utf8");
      assert(result.startsWith(GENERATED_HEADER), "Generated header should be prepended");
      assert(!result.includes("TODO:"), "TODO lines should be removed");
      assert(result.includes("export interface User"), "Non-TODO lines should be preserved");
    } finally {
      fs.rmSync(testFile, { force: true });
    }
  });

  test("Header normalization is idempotent", () => {
    const normalized = normalizeGeneratedFile(`${GENERATED_HEADER}export const x = 1;\n`);
    assertEqual(
      normalized,
      `${GENERATED_HEADER}export const x = 1;\n`,
      "Normalization should not duplicate the generated header",
    );
  });

  test("ensureGeneratedHeader prepends the repo header", () => {
    const result = ensureGeneratedHeader("export const x = 1;\n");
    assert(result.startsWith(GENERATED_HEADER), "Header should be inserted");
    assert(result.endsWith("export const x = 1;\n"), "Original content should remain after the header");
  });

  test("Script handles directory recursively", () => {
    const subDir = path.join(tempDir, "subdir");
    fs.mkdirSync(subDir, { recursive: true });

    const file1 = path.join(tempDir, "root.ts");
    const file2 = path.join(subDir, "nested.ts");
    fs.writeFileSync(file1, "export const x = 1;", "utf8");
    fs.writeFileSync(file2, "export const y = 2;", "utf8");

    try {
      const files = findTsFiles(tempDir).map((file) => path.resolve(file));
      assertEqual(files.length, 2, "Should find both TypeScript files");
      assert(files.includes(path.resolve(file1)), "Should find root file");
      assert(files.includes(path.resolve(file2)), "Should find nested file");
    } finally {
      fs.rmSync(subDir, { recursive: true, force: true });
      fs.rmSync(file1, { force: true });
    }
  });

  test("Script processes .ts and .mts files only", () => {
    const subDir = path.join(tempDir, "subdir2");
    fs.mkdirSync(subDir, { recursive: true });

    const tsFile = path.join(tempDir, "file.ts");
    const mtsFile = path.join(subDir, "file.mts");
    const jsFile = path.join(subDir, "file.js");
    const jsonFile = path.join(tempDir, "file.json");

    fs.writeFileSync(tsFile, "export const x = 1;", "utf8");
    fs.writeFileSync(mtsFile, "export const y = 2;", "utf8");
    fs.writeFileSync(jsFile, "const z = 3;", "utf8");
    fs.writeFileSync(jsonFile, "{}", "utf8");

    try {
      const files = findTsFiles(tempDir).map((file) => path.resolve(file));
      assertEqual(files.length, 2, "Should only find .ts and .mts files");
      assert(files.includes(path.resolve(tsFile)), "Should include .ts file");
      assert(files.includes(path.resolve(mtsFile)), "Should include .mts file");
      assert(!files.includes(path.resolve(jsFile)), "Should not include .js file");
      assert(!files.includes(path.resolve(jsonFile)), "Should not include .json file");
    } finally {
      fs.rmSync(subDir, { recursive: true, force: true });
      fs.rmSync(tsFile, { force: true });
      fs.rmSync(jsonFile, { force: true });
    }
  });

  test("Files without TODOs still gain the generated header", () => {
    const testFile = path.join(tempDir, "plain.ts");
    fs.writeFileSync(testFile, "export const x = 1;\n", "utf8");

    try {
      const modified = stripTodosFromFile(testFile);
      const result = fs.readFileSync(testFile, "utf8");
      assert(modified, "Header insertion should count as a modification");
      assert(result.startsWith(GENERATED_HEADER), "Header should still be inserted without TODOs");
    } finally {
      fs.rmSync(testFile, { force: true });
    }
  });

  await runAsyncTest("Script displays help on --help flag", async () => {
    const testScript = path.join(__dirname, "strip_openapi_todos.test.mjs");

    await new Promise((resolve, reject) => {
      const child = spawn("node", [testScript, "--help"]);
      let stdout = "";
      let stderr = "";

      child.stdout.on("data", (data) => {
        stdout += data.toString();
      });
      child.stderr.on("data", (data) => {
        stderr += data.toString();
      });
      child.on("close", (code) => {
        try {
          assertEqual(code, 0, "Help command should exit with code 0");
          assert(stdout.includes("Usage:"), "Help should contain usage information");
          assertEqual(stderr, "", "Help should not write to stderr");
          resolve();
        } catch (error) {
          reject(error);
        }
      });
    });
  });

  fs.rmSync(tempDir, { recursive: true, force: true });

  if (testsFailed > 0) {
    console.error(`\nTests: ${testsPassed} passed, ${testsFailed} failed`);
    process.exit(1);
  }

  console.log(`\nTests: ${testsPassed} passed, ${testsFailed} failed`);
  process.exit(0);
}

runTests().catch((error) => {
  console.error(error);
  process.exit(1);
});
