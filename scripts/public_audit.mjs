#!/usr/bin/env node

/**
 * Purpose: Prevent accidental publication of sensitive or low-signal repository content.
 * Responsibilities: Scan tracked files for leak indicators (absolute paths, placeholder contacts),
 *   high-confidence secret patterns, tracked local/build artifacts that should not be committed, and
 *   high-risk history residue in the current branch.
 * Scope: Repository metadata/docs, tracked file paths, and branch history hygiene checks only; does
 *   not lint source logic.
 * Usage: node scripts/public_audit.mjs [--json] [--branch <ref>] [--no-history] [--help]
 * Invariants/Assumptions: Runs inside a git repository with git available on PATH.
 */

import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const EXIT_SUCCESS = 0;
const EXIT_FAILURE = 1;
const EXIT_USAGE = 2;
const DEFAULT_BRANCH_REF = 'HEAD';
const MAX_HISTORY_COMMITS_PER_FILE = 300;

const CONTENT_RULES = [
	{
		id: 'abs-path',
		description: 'Absolute local machine path detected',
		pattern: /(?:\/Users\/[^\s"'<>]+|\/home\/[^\s"'<>]+|file:\/\/\/Users\/[^\s"'<>]+|[A-Za-z]:\\Users\\[^\s"'<>]+)/,
	},
	{
		id: 'placeholder',
		description: 'Public-facing placeholder text detected',
		pattern:
			/(CONTACT_EMAIL_TO_BE_UPDATED_WHEN_PUBLIC|contact details to be updated when project goes public|private contact \(contact details to be added when project goes public\))/i,
	},
];

const SECRET_RULES = [
	{
		id: 'secret-openai',
		description: 'Potential OpenAI API key detected',
		pattern: /\bsk-(?:proj-)?[A-Za-z0-9]{20,}\b/,
	},
	{
		id: 'secret-github',
		description: 'Potential GitHub token detected',
		pattern: /\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})\b/,
	},
	{
		id: 'secret-aws',
		description: 'Potential AWS access key detected',
		pattern: /\b(?:AKIA|ASIA)[A-Z0-9]{16}\b/,
	},
	{
		id: 'secret-slack',
		description: 'Potential Slack token detected',
		pattern: /\bxox[baprs]-[A-Za-z0-9-]{10,}\b/,
	},
	{
		id: 'secret-private-key',
		description: 'Private key material detected',
		pattern: /-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----/,
	},
];

const CONTENT_SCAN_PATHS = [
	/^[^/]+\.md$/,
	/^docs\/.*\.md$/,
	/^\.github\/.*\.(md|ya?ml)$/,
	/^\.env\.example$/,
	/^web\/\.env\.example$/,
	/^extensions\/\.env\.example$/,
];

const SECRET_SCAN_PATHS = [
	/^[^/]+\.(?:md|json|ya?ml|toml|ini|conf|sh|zsh|bash|env|txt)$/,
	/^(?:docs|scripts|cmd|internal|api|web|extensions|\.github)\/.*\.(?:go|ts|tsx|js|mjs|cjs|json|ya?ml|md|sh|sql|txt)$/,
	/^(?:\.env\.example|web\/\.env\.example|extensions\/\.env\.example)$/,
];

const TRACKED_PATH_RULES = [
	{
		id: 'tracked-artifact',
		description: 'Tracked local data directory should be ignored',
		pattern: /^\.data\//,
	},
	{
		id: 'tracked-artifact',
		description: 'Tracked agent cache/log directory should be ignored',
		pattern: /^\.ralph\//,
	},
	{
		id: 'tracked-artifact',
		description: 'Tracked dependency directory should be ignored',
		pattern: /(^|\/)node_modules\//,
	},
	{
		id: 'tracked-artifact',
		description: 'Tracked build output directory should be ignored',
		pattern: /(^|\/)dist\//,
	},
	{
		id: 'tracked-artifact',
		description: 'Tracked local output directory should be ignored',
		pattern: /^out\//,
	},
	{
		id: 'tracked-artifact',
		description: 'Tracked OpenAPI codegen error logs should be ignored',
		pattern: /^web\/openapi-ts-error-.*\.log$/,
	},
];

const HISTORY_PATH_RULES = [
	{
		id: 'history-artifact',
		description: 'Tracked agent cache/log directory exists in branch history',
		pathspec: '.ralph',
	},
	{
		id: 'history-artifact',
		description: 'Tracked local output directory exists in branch history',
		pathspec: 'out',
	},
	{
		id: 'history-artifact',
		description: 'Tracked local data directory exists in branch history',
		pathspec: '.data',
	},
];

const HISTORY_CONTENT_SCAN_FILES = ['docs/landscape.md'];

function usage() {
	console.log(`
Public repository audit for leak prevention and hygiene.

Usage:
  node scripts/public_audit.mjs [options]

Options:
  -h, --help    Show this help message and exit
  --json        Emit machine-readable JSON output
  --branch REF  Scan history for this ref (default: HEAD)
  --no-history  Skip history checks (faster)

Examples:
  # Human-readable output
  node scripts/public_audit.mjs

  # JSON output for automation
  node scripts/public_audit.mjs --json

  # Scan a specific branch history
  node scripts/public_audit.mjs --branch main

  # Skip history checks for a quick tracked-files-only run
  node scripts/public_audit.mjs --no-history

Exit codes:
  0   No findings
  1   Findings detected or runtime failure
  2   Usage error (unknown argument)
`);
}

function parseArgs(argv) {
	const options = {
		json: false,
		branch: DEFAULT_BRANCH_REF,
		includeHistory: true,
	};

	for (let index = 0; index < argv.length; index++) {
		const arg = argv[index];
		if (arg === '--json') {
			options.json = true;
			continue;
		}

		if (arg === '--branch') {
			const branch = argv[index + 1];
			if (!branch || branch.startsWith('-')) {
				console.error('Error: --branch requires a ref value');
				usage();
				process.exit(EXIT_USAGE);
			}
			options.branch = branch;
			index++;
			continue;
		}

		if (arg === '--no-history') {
			options.includeHistory = false;
			continue;
		}

		if (arg === '--help' || arg === '-h') {
			usage();
			process.exit(EXIT_SUCCESS);
		}

		console.error(`Error: Unknown option: ${arg}`);
		usage();
		process.exit(EXIT_USAGE);
	}

	return options;
}

function repoRoot() {
	return execFileSync('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8' }).trim();
}

function trackedFiles(root) {
	const output = execFileSync('git', ['-C', root, 'ls-files', '-z'], { encoding: 'utf8' });
	return output.split('\0').filter(Boolean);
}

function branchExists(root, ref) {
	try {
		execFileSync('git', ['-C', root, 'rev-parse', '--verify', ref], {
			encoding: 'utf8',
			stdio: ['ignore', 'pipe', 'ignore'],
		});
		return true;
	} catch {
		return false;
	}
}

function shouldScanContent(filePath) {
	return CONTENT_SCAN_PATHS.some(pattern => pattern.test(filePath));
}

function shouldScanSecrets(filePath) {
	return SECRET_SCAN_PATHS.some(pattern => pattern.test(filePath));
}

function isTrackedEnvFile(filePath) {
	if (filePath.endsWith('.env.example')) {
		return false;
	}

	if (filePath === '.env' || filePath.startsWith('.env.')) {
		return true;
	}

	if (filePath.endsWith('/.env')) {
		return true;
	}

	return /\/\.env\./.test(filePath);
}

function findPathFindings(filePath) {
	const findings = [];

	for (const rule of TRACKED_PATH_RULES) {
		if (!rule.pattern.test(filePath)) {
			continue;
		}

		findings.push({
			ruleId: rule.id,
			description: rule.description,
			file: filePath,
			line: null,
			match: filePath,
		});
	}

	if (isTrackedEnvFile(filePath)) {
		findings.push({
			ruleId: 'tracked-artifact',
			description: 'Tracked environment file should be ignored',
			file: filePath,
			line: null,
			match: filePath,
		});
	}

	return findings;
}

function findRuleFindings(filePath, content, rules) {
	const findings = [];
	const lines = content.split(/\r?\n/);

	for (let index = 0; index < lines.length; index++) {
		const line = lines[index];

		for (const rule of rules) {
			const match = line.match(rule.pattern);
			if (!match) {
				continue;
			}

			findings.push({
				ruleId: rule.id,
				description: rule.description,
				file: filePath,
				line: index + 1,
				match: match[0],
			});
		}
	}

	return findings;
}

function findContentFindings(filePath, content) {
	return findRuleFindings(filePath, content, CONTENT_RULES);
}

function findSecretFindings(filePath, content) {
	return findRuleFindings(filePath, content, SECRET_RULES);
}

function historyHasPath(root, ref, pathspec) {
	try {
		const output = execFileSync(
			'git',
			['-C', root, 'rev-list', '--max-count=1', ref, '--', pathspec],
			{
				encoding: 'utf8',
				stdio: ['ignore', 'pipe', 'ignore'],
			},
		).trim();
		return output.length > 0;
	} catch {
		return false;
	}
}

function findHistoryPathFindings(root, ref) {
	const findings = [];

	for (const rule of HISTORY_PATH_RULES) {
		if (!historyHasPath(root, ref, rule.pathspec)) {
			continue;
		}

		findings.push({
			ruleId: rule.id,
			description: rule.description,
			file: rule.pathspec,
			line: null,
			match: ref,
		});
	}

	return findings;
}

function listHistoryRevisionsForFile(root, ref, filePath) {
	try {
		const output = execFileSync(
			'git',
			['-C', root, 'rev-list', `--max-count=${MAX_HISTORY_COMMITS_PER_FILE}`, ref, '--', filePath],
			{
				encoding: 'utf8',
				stdio: ['ignore', 'pipe', 'ignore'],
			},
		).trim();
		return output.split('\n').filter(Boolean);
	} catch {
		return [];
	}
}

function readHistoryFile(root, revision, filePath) {
	try {
		return execFileSync('git', ['-C', root, 'show', `${revision}:${filePath}`], {
			encoding: 'utf8',
			stdio: ['ignore', 'pipe', 'ignore'],
		});
	} catch {
		return null;
	}
}

function findHistoryContentFindings(root, ref) {
	const findings = [];

	for (const filePath of HISTORY_CONTENT_SCAN_FILES) {
		const revisions = listHistoryRevisionsForFile(root, ref, filePath);

		for (const revision of revisions) {
			const content = readHistoryFile(root, revision, filePath);
			if (content === null) {
				continue;
			}

			const revisionFindings = [
				...findContentFindings(filePath, content),
				...findSecretFindings(filePath, content),
			];
			for (const finding of revisionFindings) {
				findings.push({
					...finding,
					description: `${finding.description} (history)`,
					match: `${revision.slice(0, 12)}:${finding.match}`,
				});
			}
		}
	}

	return findings;
}

function dedupeFindings(findings) {
	const keys = new Set();
	const deduped = [];

	for (const finding of findings) {
		const key = `${finding.ruleId}|${finding.file}|${finding.line ?? 'na'}|${finding.match}`;
		if (keys.has(key)) {
			continue;
		}
		keys.add(key);
		deduped.push(finding);
	}

	return deduped;
}

function runAudit(options = { branch: DEFAULT_BRANCH_REF, includeHistory: true }) {
	const root = repoRoot();
	const files = trackedFiles(root);
	const findings = [];

	if (options.includeHistory) {
		if (!branchExists(root, options.branch)) {
			throw new Error(`Unknown git ref for --branch: ${options.branch}`);
		}
		findings.push(...findHistoryPathFindings(root, options.branch));
		findings.push(...findHistoryContentFindings(root, options.branch));
	}

	for (const filePath of files) {
		findings.push(...findPathFindings(filePath));

		const scanContent = shouldScanContent(filePath);
		const scanSecrets = shouldScanSecrets(filePath);
		if (!scanContent && !scanSecrets) {
			continue;
		}

		try {
			const content = readFileSync(join(root, filePath), 'utf8');
			if (scanContent) {
				findings.push(...findContentFindings(filePath, content));
			}
			if (scanSecrets) {
				findings.push(...findSecretFindings(filePath, content));
			}
		} catch {
			// Ignore unreadable files; this audit only targets plaintext tracked files.
		}
	}

	const uniqueFindings = dedupeFindings(findings);

	uniqueFindings.sort((a, b) => {
		if (a.file === b.file) {
			return (a.line ?? 0) - (b.line ?? 0);
		}

		return a.file.localeCompare(b.file);
	});

	return uniqueFindings;
}

function printHuman(findings) {
	if (findings.length === 0) {
		console.log('public_audit: no findings detected in tracked files.');
		return;
	}

	console.error(`public_audit: ${findings.length} finding(s) detected:`);
	for (const finding of findings) {
		const lineSuffix = finding.line === null ? '' : `:${finding.line}`;
		console.error(
			`- [${finding.ruleId}] ${finding.file}${lineSuffix} (${finding.description}) => ${finding.match}`,
		);
	}
}

function printJSON(findings) {
	const payload = {
		ok: findings.length === 0,
		findings,
	};
	console.log(JSON.stringify(payload, null, 2));
}

function main(argv = process.argv.slice(2)) {
	const options = parseArgs(argv);

	let findings;
	try {
		findings = runAudit({ branch: options.branch, includeHistory: options.includeHistory });
	} catch (error) {
		if (options.json) {
			console.log(
				JSON.stringify(
					{
						ok: false,
						error: error instanceof Error ? error.message : String(error),
					},
					null,
					2,
				),
			);
		} else {
			console.error(`public_audit: runtime failure: ${error instanceof Error ? error.message : String(error)}`);
		}
		process.exit(EXIT_FAILURE);
	}

	if (options.json) {
		printJSON(findings);
	} else {
		printHuman(findings);
	}

	process.exit(findings.length === 0 ? EXIT_SUCCESS : EXIT_FAILURE);
}

export {
	CONTENT_RULES,
	CONTENT_SCAN_PATHS,
	DEFAULT_BRANCH_REF,
	HISTORY_CONTENT_SCAN_FILES,
	HISTORY_PATH_RULES,
	SECRET_RULES,
	SECRET_SCAN_PATHS,
	dedupeFindings,
	findHistoryContentFindings,
	findHistoryPathFindings,
	findRuleFindings,
	findSecretFindings,
	TRACKED_PATH_RULES,
	branchExists,
	findContentFindings,
	findPathFindings,
	isTrackedEnvFile,
	listHistoryRevisionsForFile,
	main,
	parseArgs,
	readHistoryFile,
	runAudit,
	shouldScanContent,
	shouldScanSecrets,
};

if (import.meta.url === `file://${process.argv[1]}`) {
	main();
}
