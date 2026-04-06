#!/usr/bin/env node

/**
 * Purpose: Validate scripts/public_audit.mjs behavior with deterministic unit-level checks.
 * Responsibilities: Verify path/content rule matching, content scan allowlist, and CLI help behavior.
 * Scope: Local script function correctness only; does not perform full repository integration testing.
 * Usage: node scripts/public_audit.test.mjs [--help]
 * Invariants/Assumptions: scripts/public_audit.mjs exports testable helpers.
 */

import { spawn } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

function usage() {
	console.log(`
Test runner for scripts/public_audit.mjs.

Usage:
  node scripts/public_audit.test.mjs [options]

Options:
  -h, --help    Show this help message and exit

Examples:
  node scripts/public_audit.test.mjs
  node scripts/public_audit.test.mjs --help

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
	const module = await import('./public_audit.mjs');
	const scriptPath = join(__dirname, 'public_audit.mjs');

	await test('isTrackedEnvFile returns true for tracked env files', () => {
		assert(module.isTrackedEnvFile('.env'), 'Expected .env to be tracked env file');
		assert(module.isTrackedEnvFile('.env.local'), 'Expected .env.local to be tracked env file');
		assert(module.isTrackedEnvFile('web/.env'), 'Expected web/.env to be tracked env file');
		assert(module.isTrackedEnvFile('web/.env.local'), 'Expected web/.env.local to be tracked env file');
	});

	await test('isTrackedEnvFile excludes example env files', () => {
		assertEqual(module.isTrackedEnvFile('.env.example'), false, 'Expected .env.example to be excluded');
		assertEqual(module.isTrackedEnvFile('web/.env.example'), false, 'Expected web/.env.example to be excluded');
	});

	await test('shouldScanContent only scans expected metadata/docs paths', () => {
		assert(module.shouldScanContent('docs/usage.md'), 'Expected docs markdown to be included');
		assert(module.shouldScanContent('docs/evidence/run/summary.json'), 'Expected docs evidence JSON to be included');
		assert(module.shouldScanContent('docs/evidence/run/output.txt'), 'Expected docs evidence text to be included');
		assert(module.shouldScanContent('docs/evidence/run/server.log'), 'Expected docs evidence logs to be included');
		assert(module.shouldScanContent('docs/evidence/run/index.html'), 'Expected docs evidence HTML to be included');
		assert(module.shouldScanContent('README.md'), 'Expected README to be included');
		assert(module.shouldScanContent('CHANGELOG.md'), 'Expected root markdown docs to be included');
		assert(module.shouldScanContent('.github/PULL_REQUEST_TEMPLATE.md'), 'Expected .github markdown docs to be included');
		assertEqual(module.shouldScanContent('internal/cli/cli.go'), false, 'Expected source Go file to be excluded');
	});

	await test('shouldScanSecrets scans source and config surfaces', () => {
		assert(module.shouldScanSecrets('internal/cli/cli.go'), 'Expected Go source to be secret-scanned');
		assert(module.shouldScanSecrets('web/src/App.tsx'), 'Expected TSX source to be secret-scanned');
		assert(module.shouldScanSecrets('web/package.json'), 'Expected web root config to be secret-scanned');
		assert(module.shouldScanSecrets('.github/workflows/ci-pr.yml'), 'Expected workflow YAML to be secret-scanned');
		assert(module.shouldScanSecrets('docs/evidence/run/server.log'), 'Expected docs evidence logs to be secret-scanned');
		assert(module.shouldScanSecrets('docs/evidence/run/index.html'), 'Expected docs evidence HTML to be secret-scanned');
		assert(module.shouldScanSecrets('.env.example'), 'Expected .env.example to be secret-scanned');
		assertEqual(module.shouldScanSecrets('assets/logo.png'), false, 'Expected binary asset path to be excluded');
	});

	await test('findPathFindings catches tracked artifact directories', () => {
		const findings = module.findPathFindings('.ralph/done.json');
		assert(findings.length > 0, 'Expected at least one finding for .ralph path');
		assertEqual(findings[0].ruleId, 'tracked-artifact', 'Expected tracked-artifact rule ID');
	});

	await test('findPathFindings catches tracked binary artifacts', () => {
		const findings = module.findPathFindings('bin/spartan');
		assert(findings.length > 0, 'Expected at least one finding for bin path');
		assert(
			findings.some(finding => finding.description.includes('compiled binary')),
			'Expected compiled binary finding description',
		);
	});

	await test('findPathFindings catches tracked Finder metadata', () => {
		const findings = module.findPathFindings('web/.DS_Store');
		assert(findings.length > 0, 'Expected at least one finding for .DS_Store path');
		assert(
			findings.some(finding => finding.description.includes('Finder metadata')),
			'Expected Finder metadata finding description',
		);
	});

	await test('findContentFindings catches absolute paths and placeholders', () => {
		const findings = module.findContentFindings(
			'docs/example.md',
			'Path /Users/tester/private.txt\nScratch /tmp/private.txt\nChrome /Applications/Google Chrome.app/Contents/MacOS/Google Chrome\nCONTACT_EMAIL_TO_BE_UPDATED_WHEN_PUBLIC',
		);
		assertEqual(findings.length, 4, 'Expected all absolute-path variants and placeholder findings');
	});

	await test('findContentFindings ignores URL path segments that resemble local paths', () => {
		const findings = module.findContentFindings(
			'docs/example.md',
			'See https://example.com/tmp/report and https://example.com/Applications/Preview for hosted artifacts.',
		);
		assertEqual(findings.length, 0, 'Expected hosted URL path segments to be ignored');
	});

	await test('findContentFindings ignores route-like tmp/application paths in markdown and route fields', () => {
		const findings = module.findContentFindings(
			'docs/example.md',
			'[report](/tmp/report)\n{"path":"/tmp/report"}\n{"url": "/Applications/Preview"}',
		);
		assertEqual(findings.length, 0, 'Expected route-like tmp/application paths to be ignored in route contexts');
	});

	await test('findContentFindings catches quoted tmp/application artifact paths in evidence-like JSON', () => {
		const findings = module.findContentFindings(
			'docs/example.md',
			'{"artifactPath":"/tmp/run/export.md","chromeBinary":"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}',
		);
		assertEqual(findings.length, 2, 'Expected evidence-style quoted machine paths to be detected');
	});

	await test('findContentFindings catches absolute paths inside markdown inline code', () => {
		const findings = module.findContentFindings(
			'docs/example.md',
			'Inline code: `/Users/alice/private.txt`\nMore inline code: `/tmp/build.log`',
		);
		assertEqual(findings.length, 2, 'Expected inline-code absolute paths to be detected');
	});

	await test('shouldIgnoreAbsolutePathMatch ignores only route-like ambiguous paths', () => {
		const tmpRouteField = '{"path":"/tmp/report"}';
		const applicationRouteLink = '[report](/Applications/Preview)';
		const artifactPathField = '{"artifactPath":"/tmp/run/export.md"}';
		const usersPathField = '{"path":"/Users/alice/private.txt"}';

		assertEqual(
			module.shouldIgnoreAbsolutePathMatch(
				tmpRouteField,
				'/tmp/report',
				tmpRouteField.indexOf('/tmp/report'),
			),
			true,
			'Expected shallow /tmp route field to be ignored',
		);
		assertEqual(
			module.shouldIgnoreAbsolutePathMatch(
				applicationRouteLink,
				'/Applications/Preview',
				applicationRouteLink.indexOf('/Applications/Preview'),
			),
			true,
			'Expected markdown-link application route to be ignored',
		);
		assertEqual(
			module.shouldIgnoreAbsolutePathMatch(
				artifactPathField,
				'/tmp/run/export.md',
				artifactPathField.indexOf('/tmp/run/export.md'),
			),
			false,
			'Expected evidence-style tmp artifact path to remain reportable',
		);
		assertEqual(
			module.shouldIgnoreAbsolutePathMatch('`/tmp/report`', '/tmp/report', 1),
			false,
			'Expected inline-code /tmp path to be kept',
		);
		assertEqual(
			module.shouldIgnoreAbsolutePathMatch(
				usersPathField,
				'/Users/alice/private.txt',
				usersPathField.indexOf('/Users/alice/private.txt'),
			),
			false,
			'Expected /Users path to remain reportable',
		);
	});

	await test('findContentFindings keeps scanning after an ignored route-like match on the same line', () => {
		const findings = module.findContentFindings(
			'docs/example.md',
			'[report](/tmp/report) leaked /Users/alice/private.txt',
		);
		assertEqual(findings.length, 1, 'Expected later real machine path to still be detected');
		assertEqual(findings[0].match, '/Users/alice/private.txt', 'Expected the real machine path to be reported');
	});

	await test('findSecretFindings catches high-confidence secret patterns', () => {
		const token = `sk-proj-${'ABCDEFGHIJKLMNOPQRSTUVWXYZ12345'}`;
		const findings = module.findSecretFindings(
			'internal/example.go',
			`OPENAI_API_KEY=${token}`,
		);
		assertEqual(findings.length, 1, 'Expected one secret finding');
		assertEqual(findings[0].ruleId, 'secret-openai', 'Expected OpenAI secret rule ID');
	});

	await test('findSecretFindings catches secrets in evidence logs', () => {
		const token = ['ghp_', 'ABCDEFGHIJKLMNOPQRSTUVWXYZ123456'].join('');
		const findings = module.findSecretFindings(
			'docs/evidence/run/server.log',
			`authorization failed for token ${token}`,
		);
		assertEqual(findings.length, 1, 'Expected one secret finding in log content');
		assertEqual(findings[0].ruleId, 'secret-github', 'Expected GitHub secret rule ID');
	});

	await test('parseArgs supports branch and history toggles', () => {
		const options = module.parseArgs(['--json', '--branch', 'main', '--no-history']);
		assertEqual(options.json, true, 'Expected --json to enable JSON output');
		assertEqual(options.branch, 'main', 'Expected --branch to set branch ref');
		assertEqual(options.includeHistory, false, 'Expected --no-history to disable history scanning');
	});

	await test('parseArgs defaults to HEAD history scanning', () => {
		const options = module.parseArgs([]);
		assertEqual(options.branch, module.DEFAULT_BRANCH_REF, 'Expected default branch ref to be HEAD');
		assertEqual(options.includeHistory, true, 'Expected history scanning enabled by default');
	});

	await test('branchExists returns expected values', () => {
		const root = process.cwd();
		assert(module.branchExists(root, 'HEAD'), 'Expected HEAD ref to exist');
		assertEqual(module.branchExists(root, 'definitely-not-a-real-ref'), false, 'Expected fake ref to fail');
	});

	await test('HISTORY_PATH_RULES includes compiled binary path scanning', () => {
		assert(
			module.HISTORY_PATH_RULES.some(rule => rule.pathspec === 'bin'),
			'Expected history path rules to include bin pathspec',
		);
	});

	await test('dedupeFindings removes exact duplicates', () => {
		const input = [
			{ ruleId: 'x', file: 'a.md', line: 1, match: 'abc' },
			{ ruleId: 'x', file: 'a.md', line: 1, match: 'abc' },
			{ ruleId: 'x', file: 'a.md', line: 2, match: 'abc' },
		];
		const deduped = module.dedupeFindings(input);
		assertEqual(deduped.length, 2, 'Expected duplicate finding to be removed');
	});

	await test('listHistoryRevisionsForFile returns revisions for tracked files', () => {
		const root = process.cwd();
		const revisions = module.listHistoryRevisionsForFile(root, 'HEAD', 'README.md');
		assert(revisions.length > 0, 'Expected at least one revision for README.md');
	});

	await test('public_audit.mjs --help exits successfully and prints usage', async () => {
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
