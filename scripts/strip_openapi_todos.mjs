#!/usr/bin/env node

/**
 * Strip TODO comment lines from generated OpenAPI TypeScript output.
 *
 * Usage:
 *   node strip_openapi_todos.mjs --path <dir>
 *
 * Example:
 *   node strip_openapi_todos.mjs --path web/src/api
 */

import fs from 'fs';
import path from 'path';

function usage() {
	console.log(`
Strip TODO comment lines from generated OpenAPI TypeScript output.

Usage:
  node scripts/strip_openapi_todos.mjs --path <dir>

Example:
  node scripts/strip_openapi_todos.mjs --path web/src/api
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
		} else if (entry.isFile() && (entry.name.endsWith('.ts') || entry.name.endsWith('.mts'))) {
			files.push(fullPath);
		}
	}

	return files;
}

function stripTodosFromFile(filePath) {
	const content = fs.readFileSync(filePath, 'utf8');
	const lines = content.split(/\r?\n/);
	const filteredLines = lines.filter(line => !line.match(/^.*TODO:.*$/));

	if (filteredLines.length === lines.length) {
		return false;
	}

	const newContent = filteredLines.join('\n');
	fs.writeFileSync(filePath, newContent, 'utf8');
	return true;
}

function main() {
	const args = process.argv.slice(2);
	let targetPath = '';

	for (let i = 0; i < args.length; i++) {
		if (args[i] === '--help' || args[i] === '-h') {
			usage();
			process.exit(0);
		} else if (args[i] === '--path') {
			if (i + 1 >= args.length) {
				showError('Error: --path requires an argument');
				usage();
				process.exit(2);
			}
			targetPath = args[i + 1];
			i++;
		} else {
			showError(`Error: Unknown argument: ${args[i]}`);
			usage();
			process.exit(2);
		}
	}

	if (!targetPath) {
		showError('Error: Missing --path');
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
		const tsFiles = findTsFiles(targetPath);
		let modifiedCount = 0;

		for (const file of tsFiles) {
			if (stripTodosFromFile(file)) {
				modifiedCount++;
			}
		}

		process.exit(0);
	} catch (error) {
		showError(`Error: ${error.message}`);
		process.exit(1);
	}
}

export { usage, showError, findTsFiles, stripTodosFromFile, main };

if (import.meta.url === `file://${process.argv[1]}`) {
	main();
}
