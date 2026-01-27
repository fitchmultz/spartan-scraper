#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

let testsPassed = 0;
let testsFailed = 0;

function test(name, fn) {
	try {
		fn();
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
		throw new Error(message || 'Assertion failed');
	}
}

function assertEqual(actual, expected, message) {
	if (actual !== expected) {
		throw new Error(message || `Expected ${expected}, got ${actual}`);
	}
}

async function runTests() {
	const tempDir = path.join(__dirname, '.test-tmp');

	if (fs.existsSync(tempDir)) {
		fs.rmSync(tempDir, { recursive: true, force: true });
	}
	fs.mkdirSync(tempDir, { recursive: true });

	const { findTsFiles, stripTodosFromFile } = await import('./strip_openapi_todos.mjs');

	test('Script removes TODO lines from files', () => {
			const testFile = path.join(tempDir, 'test.ts');
			fs.writeFileSync(testFile, `
export interface User {
  name: string;
  age: number;
}
// TODO: implement validation
export function validate(user: User): boolean {
  return user.age > 0;
}
// TODO: add email validation
`, 'utf8');

			try {
				stripTodosFromFile(testFile);

				const result = fs.readFileSync(testFile, 'utf8');
				assert(!result.includes('TODO:'), 'TODO lines should be removed');
				assert(result.includes('export interface User'), 'Non-TODO lines should be preserved');
				assert(result.includes('export function validate'), 'Non-TODO lines should be preserved');
			} finally {
				fs.rmSync(testFile, { force: true });
			}
		});

		test('Script preserves non-TODO lines', () => {
			const testFile = path.join(tempDir, 'test2.ts');
			const originalContent = `
export interface Product {
  id: string;
  name: string;
  price: number;
}

export function getPrice(product: Product): number {
  return product.price;
}
`;
			fs.writeFileSync(testFile, originalContent, 'utf8');

			try {
				stripTodosFromFile(testFile);

				const result = fs.readFileSync(testFile, 'utf8');
				assertEqual(result, originalContent, 'Non-TODO files should remain unchanged');
			} finally {
				fs.rmSync(testFile, { force: true });
			}
		});

		test('Script handles directory recursively', () => {
			const subDir = path.join(tempDir, 'subdir');
			fs.mkdirSync(subDir, { recursive: true });

			const file1 = path.join(tempDir, 'root.ts');
			const file2 = path.join(subDir, 'nested.ts');

			fs.writeFileSync(file1, '// TODO: fix this\nexport const x = 1;', 'utf8');
			fs.writeFileSync(file2, '// TODO: fix that\nexport const y = 2;', 'utf8');

			try {
				const files = findTsFiles(tempDir);
				const resolvedFiles = files.map(f => path.resolve(f));
				assert(files.length === 2, 'Should find both TypeScript files');
				assert(resolvedFiles.includes(path.resolve(file1)), 'Should find root file');
				assert(resolvedFiles.includes(path.resolve(file2)), 'Should find nested file');
			} finally {
				fs.rmSync(subDir, { recursive: true, force: true });
				fs.rmSync(file1, { force: true });
			}
		});

		test('Script processes .ts and .mts files only', () => {
			const subDir = path.join(tempDir, 'subdir2');
			fs.mkdirSync(subDir, { recursive: true });

			const tsFile = path.join(tempDir, 'file.ts');
			const mtsFile = path.join(subDir, 'file.mts');
			const jsFile = path.join(subDir, 'file.js');
			const jsonFile = path.join(tempDir, 'file.json');

			fs.writeFileSync(tsFile, '// TODO\nexport const x = 1;', 'utf8');
			fs.writeFileSync(mtsFile, '// TODO\nexport const y = 2;', 'utf8');
			fs.writeFileSync(jsFile, '// TODO\nconst z = 3;', 'utf8');
			fs.writeFileSync(jsonFile, '{}', 'utf8');

			try {
				const files = findTsFiles(tempDir);
				const resolvedFiles = files.map(f => path.resolve(f));
				assertEqual(files.length, 2, 'Should only find .ts and .mts files');
				assert(resolvedFiles.includes(path.resolve(tsFile)), 'Should include .ts file');
				assert(resolvedFiles.includes(path.resolve(mtsFile)), 'Should include .mts file');
				assert(!resolvedFiles.includes(path.resolve(jsFile)), 'Should not include .js file');
				assert(!resolvedFiles.includes(path.resolve(jsonFile)), 'Should not include .json file');
			} finally {
				fs.rmSync(subDir, { recursive: true, force: true });
				fs.rmSync(tsFile, { force: true });
				fs.rmSync(jsonFile, { force: true });
			}
		});

		test('Script handles files with no TODOs', () => {
			const testFile = path.join(tempDir, 'test3.ts');
			const content = 'export const x = 1;';
			fs.writeFileSync(testFile, content, 'utf8');

			const modified = stripTodosFromFile(testFile);

			assert(!modified, 'Should return false when no changes made');
			assertEqual(fs.readFileSync(testFile, 'utf8'), content, 'Content should remain unchanged');
		});

	fs.rmSync(tempDir, { recursive: true, force: true });

	console.log(`\nTests: ${testsPassed} passed, ${testsFailed} failed`);
	process.exit(testsFailed > 0 ? 1 : 0);
}

runTests().catch(error => {
	console.error('Test suite error:', error);
	process.exit(1);
});
