#!/usr/bin/env node

/**
 * Purpose: Validate scripts/go_transitive_override_audit.mjs parsing and CLI behavior.
 * Responsibilities: Check argument handling, outdated-module line parsing, requirement lookup,
 *   and help output without mutating repository state.
 * Scope: Unit-level script correctness only; does not perform a full live override audit.
 * Usage: node scripts/go_transitive_override_audit.test.mjs [--help]
 * Invariants/Assumptions: scripts/go_transitive_override_audit.mjs exports testable helpers.
 */

import { spawn } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

function usage() {
	console.log(`
Test runner for scripts/go_transitive_override_audit.mjs.

Usage:
  node scripts/go_transitive_override_audit.test.mjs [options]

Options:
  -h, --help    Show this help message and exit

Examples:
  node scripts/go_transitive_override_audit.test.mjs
  node scripts/go_transitive_override_audit.test.mjs --help

Exit codes:
  0   All tests passed
  1   One or more tests failed
  2   Usage error
`);
}

const args = process.argv.slice(2);
if (args.includes('--help') || args.includes('-h')) {
	usage();
	process.exit(0);
}

for (const arg of args) {
	console.error(`Error: Unknown option: ${arg}`);
	usage();
	process.exit(2);
}

let passed = 0;
let failed = 0;

function assert(condition, message) {
	if (!condition) {
		throw new Error(message);
	}
}

function assertEqual(actual, expected, message) {
	if (actual !== expected) {
		throw new Error(`${message} (expected: ${expected}, actual: ${actual})`);
	}
}

async function test(name, fn) {
	try {
		await fn();
		passed++;
		console.log(`✓ ${name}`);
	} catch (error) {
		failed++;
		console.error(`✗ ${name}`);
		console.error(`  ${error instanceof Error ? error.message : String(error)}`);
	}
}

function runChild(scriptPath, childArgs = []) {
	return new Promise((resolve, reject) => {
		const child = spawn('node', [scriptPath, ...childArgs]);
		let stdout = '';
		let stderr = '';

		child.stdout.on('data', data => {
			stdout += data.toString();
		});

		child.stderr.on('data', data => {
			stderr += data.toString();
		});

		child.on('error', reject);
		child.on('close', code => {
			resolve({ code, stdout, stderr });
		});
	});
}

async function run() {
	const module = await import('./go_transitive_override_audit.mjs');
	const scriptPath = join(__dirname, 'go_transitive_override_audit.mjs');

	await test('parseArgs defaults to human-readable output', () => {
		const options = module.parseArgs([]);
		assertEqual(options.json, false, 'Expected --json to default to false');
	});

	await test('parseArgs enables JSON output', () => {
		const options = module.parseArgs(['--json']);
		assertEqual(options.json, true, 'Expected --json to enable JSON output');
	});

	await test('parseOutdatedModuleLine parses replaced stale modules', () => {
		const parsed = module.parseOutdatedModuleLine(
			'github.com/coder/websocket v1.8.12 [v1.8.14] => github.com/coder/websocket v1.8.14',
		);
		assert(parsed !== null, 'Expected parsed module info');
		assertEqual(parsed.path, 'github.com/coder/websocket', 'Expected module path');
		assertEqual(parsed.selectedVersion, 'v1.8.12', 'Expected selected version');
		assertEqual(parsed.updateVersion, 'v1.8.14', 'Expected update version');
		assertEqual(parsed.replacePath, 'github.com/coder/websocket', 'Expected replace path');
		assertEqual(parsed.replaceVersion, 'v1.8.14', 'Expected replace version');
	});

	await test('parseOutdatedModuleLine ignores non-outdated lines', () => {
		assertEqual(module.parseOutdatedModuleLine('github.com/stretchr/testify v1.11.1'), null, 'Expected null for non-outdated line');
	});

	await test('findRequireVersion returns the required version for a module path', () => {
		const version = module.findRequireVersion(
			{
				Require: [
					{ Path: 'github.com/google/go-cmp', Version: 'v0.6.0' },
					{ Path: 'github.com/yuin/goldmark', Version: 'v1.4.13' },
				],
			},
			'github.com/yuin/goldmark',
		);
		assertEqual(version, 'v1.4.13', 'Expected required module version');
	});

	await test('findRequireVersion returns null when a requirement is absent', () => {
		const version = module.findRequireVersion({ Require: [] }, 'golang.org/x/xerrors');
		assertEqual(version, null, 'Expected null for missing requirement');
	});

	await test('managed override inventory remains at the documented size', () => {
		assertEqual(module.MANAGED_OVERRIDES.length, 12, 'Expected 12 managed overrides');
	});

	await test('go_transitive_override_audit.mjs --help exits successfully and prints usage', async () => {
		const result = await runChild(scriptPath, ['--help']);
		assertEqual(result.code, 0, 'Expected help exit code 0');
		assert(result.stdout.includes('Usage:'), 'Expected help output to contain Usage');
	});

	console.log(`\nTests: ${passed} passed, ${failed} failed`);
	process.exit(failed > 0 ? 1 : 0);
}

run().catch(error => {
	console.error('Test suite error:', error instanceof Error ? error.message : String(error));
	process.exit(1);
});
