#!/usr/bin/env node

/**
 * Purpose: Keep temporary Go transitive override usage explicit, current, and removable.
 * Responsibilities: Verify each managed `go.mod` replace override still points at the latest
 *   available tag, confirm the selected stale version still comes from known upstream parents,
 *   keep the root `replace` block synchronized with the managed override inventory, and fail
 *   when unmanaged stale Go modules or redundant overrides appear.
 * Scope: Root-module Go dependency metadata only (`go.mod`, `go list -m`, and upstream parent
 *   `go.mod` files); does not build or test application code.
 * Usage: node scripts/go_transitive_override_audit.mjs [--json] [--help]
 * Invariants/Assumptions: Runs from inside this repository with `git` and `go` on PATH and with
 *   `go.mod` as the root module file.
 */

import { execFileSync } from 'node:child_process';

const EXIT_SUCCESS = 0;
const EXIT_FAILURE = 1;
const EXIT_USAGE = 2;

const MANAGED_OVERRIDES = [
	{
		path: 'github.com/coder/websocket',
		selectedVersion: 'v1.8.12',
		overrideVersion: 'v1.8.14',
		parents: [
			{
				path: 'github.com/playwright-community/playwright-go',
				version: 'v0.5700.1',
				requiredVersion: 'v1.8.12',
			},
		],
	},
	{
		path: 'github.com/creack/pty',
		selectedVersion: 'v1.1.9',
		overrideVersion: 'v1.1.24',
		parents: [
			{
				path: 'github.com/kr/text',
				version: 'v0.2.0',
				requiredVersion: 'v1.1.9',
			},
		],
	},
	{
		path: 'github.com/google/go-cmp',
		selectedVersion: 'v0.6.0',
		overrideVersion: 'v0.7.0',
		parents: [
			{
				path: 'github.com/go-jose/go-jose/v3',
				version: 'v3.0.4',
				requiredVersion: 'v0.5.9',
			},
			{
				path: 'golang.org/x/tools',
				version: 'v0.43.0',
				requiredVersion: 'v0.6.0',
			},
		],
	},
	{
		path: 'github.com/ianlancetaylor/demangle',
		selectedVersion: 'v0.0.0-20250417193237-f615e6bd150b',
		overrideVersion: 'v0.0.0-20251118225945-96ee0021ea0f',
		parents: [
			{
				path: 'github.com/google/pprof',
				version: 'v0.0.0-20260302011040-a15ffb7f9dcc',
				requiredVersion: 'v0.0.0-20250417193237-f615e6bd150b',
			},
		],
	},
	{
		path: 'github.com/kr/pty',
		selectedVersion: 'v1.1.1',
		overrideVersion: 'v1.1.8',
		parents: [
			{
				path: 'github.com/kr/text',
				version: 'v0.1.0',
				requiredVersion: 'v1.1.1',
			},
		],
	},
	{
		path: 'github.com/stretchr/objx',
		selectedVersion: 'v0.5.2',
		overrideVersion: 'v0.5.3',
		parents: [
			{
				path: 'github.com/stretchr/testify',
				version: 'v1.11.1',
				requiredVersion: 'v0.5.2',
			},
		],
	},
	{
		path: 'github.com/tidwall/gjson',
		selectedVersion: 'v1.17.0',
		overrideVersion: 'v1.18.0',
		parents: [
			{
				path: 'github.com/playwright-community/playwright-go',
				version: 'v0.5700.1',
				requiredVersion: 'v1.17.0',
			},
		],
	},
	{
		path: 'github.com/tidwall/match',
		selectedVersion: 'v1.1.1',
		overrideVersion: 'v1.2.0',
		parents: [
			{
				path: 'github.com/playwright-community/playwright-go',
				version: 'v0.5700.1',
				requiredVersion: 'v1.1.1',
			},
		],
	},
	{
		path: 'github.com/yuin/goldmark',
		selectedVersion: 'v1.4.13',
		overrideVersion: 'v1.7.16',
		parents: [
			{
				path: 'golang.org/x/tools',
				version: 'v0.43.0',
				requiredVersion: 'v1.4.13',
			},
		],
	},
	{
		path: 'github.com/zeebo/assert',
		selectedVersion: 'v1.3.0',
		overrideVersion: 'v1.3.1',
		parents: [
			{
				path: 'github.com/zeebo/xxh3',
				version: 'v1.1.0',
				requiredVersion: 'v1.3.0',
			},
		],
	},
	{
		path: 'golang.org/x/telemetry',
		selectedVersion: 'v0.0.0-20260311193753-579e4da9a98c',
		overrideVersion: 'v0.0.0-20260312161427-1546bf4b83fe',
		parents: [
			{
				path: 'golang.org/x/tools',
				version: 'v0.43.0',
				requiredVersion: 'v0.0.0-20260311193753-579e4da9a98c',
			},
		],
	},
	{
		path: 'golang.org/x/xerrors',
		selectedVersion: 'v0.0.0-20190717185122-a985d3407aa7',
		overrideVersion: 'v0.0.0-20240903120638-7835f813f4da',
		parents: [
			{
				path: 'golang.org/x/tools',
				version: 'v0.0.0-20191119224855-298f0cb1881e',
				requiredVersion: 'v0.0.0-20190717185122-a985d3407aa7',
			},
		],
	},
];

function usage() {
	console.log(`
Audit managed Go transitive override replacements.

Usage:
  node scripts/go_transitive_override_audit.mjs [options]

Options:
  -h, --help    Show this help message and exit
  --json        Emit machine-readable JSON output

Examples:
  node scripts/go_transitive_override_audit.mjs
  node scripts/go_transitive_override_audit.mjs --json

Exit codes:
  0   All managed overrides are still justified and current
  1   Findings detected or runtime failure
  2   Usage error
`);
}

function parseArgs(argv) {
	const options = { json: false };

	for (const arg of argv) {
		if (arg === '--json') {
			options.json = true;
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

function runCommand(root, command, args) {
	return execFileSync(command, args, {
		cwd: root,
		encoding: 'utf8',
		stdio: ['ignore', 'pipe', 'pipe'],
	});
}

function runGo(root, args) {
	return runCommand(root, 'go', args);
}

function parseJSON(text) {
	return JSON.parse(text);
}

function goListModule(root, path) {
	return parseJSON(runGo(root, ['list', '-m', '-u', '-json', path]));
}

function goModEditJSON(root, modPath) {
	return parseJSON(runGo(root, ['mod', 'edit', '-json', modPath]));
}

function goModDownloadJSON(root, moduleSpec) {
	return parseJSON(runGo(root, ['mod', 'download', '-json', moduleSpec]));
}

function findRequireVersion(goModJSON, path) {
	if (!Array.isArray(goModJSON.Require)) {
		return null;
	}

	const requirement = goModJSON.Require.find(entry => entry.Path === path);
	return requirement?.Version ?? null;
}

function parseOutdatedModuleLine(line) {
	const trimmed = line.trim();
	if (trimmed.length === 0 || !trimmed.includes('[')) {
		return null;
	}

	const match = trimmed.match(
		/^(?<path>\S+)\s+(?<selectedVersion>\S+)\s+\[(?<updateVersion>[^\]]+)\](?:\s+=>\s+(?<replacePath>\S+)\s+(?<replaceVersion>\S+))?$/,
	);
	if (!match?.groups) {
		return null;
	}

	return {
		path: match.groups.path,
		selectedVersion: match.groups.selectedVersion,
		updateVersion: match.groups.updateVersion,
		replacePath: match.groups.replacePath ?? null,
		replaceVersion: match.groups.replaceVersion ?? null,
	};
}

function goListOutdatedModules(root) {
	const output = runGo(root, ['list', '-m', '-u', 'all']);
	return output
		.split(/\r?\n/)
		.map(parseOutdatedModuleLine)
		.filter(Boolean);
}

function rootReplaceMap(root) {
	const goMod = goModEditJSON(root, 'go.mod');
	const replaces = new Map();
	for (const replacement of goMod.Replace ?? []) {
		if (!replacement?.Old?.Path || !replacement?.New?.Version) {
			continue;
		}
		replaces.set(replacement.Old.Path, replacement.New.Version);
	}
	return replaces;
}

function findOverrideInventoryFindings(replaceMap, managedOverrides = MANAGED_OVERRIDES) {
	const findings = [];
	const managedPaths = new Set();

	for (const override of managedOverrides) {
		if (managedPaths.has(override.path)) {
			findings.push(`MANAGED_OVERRIDES contains duplicate entry for ${override.path}`);
			continue;
		}
		managedPaths.add(override.path);
	}

	for (const path of replaceMap.keys()) {
		if (!managedPaths.has(path)) {
			findings.push(`go.mod replace for ${path} is unmanaged; remove it or add a matching MANAGED_OVERRIDES entry`);
		}
	}

	for (const override of managedOverrides) {
		if (!replaceMap.has(override.path)) {
			findings.push(`MANAGED_OVERRIDES entry for ${override.path} has no matching go.mod replace`);
		}
	}

	return findings;
}

function validateManagedOverride(root, replaceMap, outdatedByPath, override) {
	const findings = [];
	const rootReplaceVersion = replaceMap.get(override.path);
	if (rootReplaceVersion !== override.overrideVersion) {
		findings.push(
			`go.mod replace for ${override.path} should point at ${override.overrideVersion}, found ${rootReplaceVersion ?? 'missing'}`,
		);
	}

	const outdated = outdatedByPath.get(override.path);
	if (!outdated) {
		findings.push(
			`${override.path} is no longer reported by 'go list -m -u all'; remove or refresh the temporary override entry`,
		);
		return findings;
	}

	if (outdated.selectedVersion !== override.selectedVersion) {
		findings.push(
			`${override.path} selected version drifted from ${override.selectedVersion} to ${outdated.selectedVersion}; re-audit the parent module graph before keeping the override`,
		);
	}
	if (outdated.updateVersion !== override.overrideVersion) {
		findings.push(
			`${override.path} latest version drifted from ${override.overrideVersion} to ${outdated.updateVersion}; update or remove the override`,
		);
	}
	if (outdated.replacePath !== override.path || outdated.replaceVersion !== override.overrideVersion) {
		findings.push(
			`${override.path} should be replaced to ${override.overrideVersion} in the selected build list, found ${outdated.replacePath ?? 'no replacement'} ${outdated.replaceVersion ?? ''}`.trim(),
		);
	}

	const moduleInfo = goListModule(root, override.path);
	if (moduleInfo.Version !== override.selectedVersion) {
		findings.push(
			`${override.path} go list selected version expected ${override.selectedVersion}, found ${moduleInfo.Version ?? 'missing'}`,
		);
	}
	if (moduleInfo.Replace?.Version !== override.overrideVersion) {
		findings.push(
			`${override.path} go list replacement expected ${override.overrideVersion}, found ${moduleInfo.Replace?.Version ?? 'missing'}`,
		);
	}
	if (moduleInfo.Update?.Version !== override.overrideVersion) {
		findings.push(
			`${override.path} latest available version expected ${override.overrideVersion}, found ${moduleInfo.Update?.Version ?? 'missing'}`,
		);
	}

	for (const parent of override.parents) {
		const parentDownload = goModDownloadJSON(root, `${parent.path}@${parent.version}`);
		const parentMod = goModEditJSON(root, parentDownload.GoMod);
		const requiredVersion = findRequireVersion(parentMod, override.path);
		if (requiredVersion !== parent.requiredVersion) {
			findings.push(
				`${parent.path}@${parent.version} should require ${override.path} ${parent.requiredVersion}, found ${requiredVersion ?? 'missing'}`,
			);
		}
	}

	return findings;
}

function runAudit() {
	const root = repoRoot();
	const replaceMap = rootReplaceMap(root);
	const outdatedModules = goListOutdatedModules(root);
	const outdatedByPath = new Map(outdatedModules.map(module => [module.path, module]));
	const managedPaths = new Set(MANAGED_OVERRIDES.map(override => override.path));
	const findings = [...findOverrideInventoryFindings(replaceMap, MANAGED_OVERRIDES)];

	for (const override of MANAGED_OVERRIDES) {
		findings.push(...validateManagedOverride(root, replaceMap, outdatedByPath, override));
	}

	for (const outdated of outdatedModules) {
		if (managedPaths.has(outdated.path)) {
			continue;
		}
		findings.push(
			`Unmanaged outdated Go module ${outdated.path} detected (${outdated.selectedVersion} -> ${outdated.updateVersion}); upgrade the parent module or add a managed override plan`,
		);
	}

	return {
		ok: findings.length === 0,
		managedOverrideCount: MANAGED_OVERRIDES.length,
		findings,
	};
}

function printHuman(result) {
	if (result.ok) {
		console.log(
			`go_transitive_override_audit: verified ${result.managedOverrideCount} managed override(s); no unmanaged stale Go modules found.`,
		);
		return;
	}

	console.error(`go_transitive_override_audit: ${result.findings.length} finding(s) detected:`);
	for (const finding of result.findings) {
		console.error(`- ${finding}`);
	}
}

function printJSON(result) {
	console.log(JSON.stringify(result, null, 2));
}

function main(argv = process.argv.slice(2)) {
	const options = parseArgs(argv);

	let result;
	try {
		result = runAudit();
	} catch (error) {
		const message = error instanceof Error ? error.message : String(error);
		if (options.json) {
			printJSON({ ok: false, findings: [], error: message, managedOverrideCount: MANAGED_OVERRIDES.length });
		} else {
			console.error(`go_transitive_override_audit: runtime failure: ${message}`);
		}
		process.exit(EXIT_FAILURE);
	}

	if (options.json) {
		printJSON(result);
	} else {
		printHuman(result);
	}

	process.exit(result.ok ? EXIT_SUCCESS : EXIT_FAILURE);
}

export {
	EXIT_FAILURE,
	EXIT_SUCCESS,
	EXIT_USAGE,
	MANAGED_OVERRIDES,
	findOverrideInventoryFindings,
	findRequireVersion,
	goListOutdatedModules,
	main,
	parseArgs,
	parseOutdatedModuleLine,
	rootReplaceMap,
	runAudit,
	usage,
	validateManagedOverride,
};

if (import.meta.url === `file://${process.argv[1]}`) {
	main();
}
