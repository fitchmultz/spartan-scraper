/**
 * Purpose: Verify bridge-side validation and normalization for AI-generated outputs.
 * Responsibilities: Cover extract normalization, template strategy acceptance, and malformed payload rejection.
 * Scope: `validation.ts` tests only.
 * Usage: Run with `pnpm --dir tools/pi-bridge test`.
 * Invariants/Assumptions: Valid templates may use selectors, JSON-LD, regex, or any mix; malformed outputs still fail fast.
 */
import assert from "node:assert/strict";
import test from "node:test";
import { normalizeExtractResult, validateTemplateResult } from "./validation.js";

test("normalizeExtractResult canonicalizes primitive field values", () => {
  const result = normalizeExtractResult(
    {
      fields: {
        title: "Orbit Widget Pro",
        price: 149,
        in_stock: true,
      },
      confidence: "0.95",
    },
    {
      provider: "kimi-coding",
      model: "k2p5",
      tokens_used: 211,
    },
  );

  assert.deepEqual(result, {
    fields: {
      title: { values: ["Orbit Widget Pro"], source: "llm" },
      price: { values: ["149"], source: "llm" },
      in_stock: { values: ["true"], source: "llm" },
    },
    confidence: 0.95,
    provider: "kimi-coding",
    model: "k2p5",
    tokens_used: 211,
  });
});

test("normalizeExtractResult preserves structured field data as rawObject", () => {
  const result = normalizeExtractResult({
    fields: {
      specs: {
        battery: "18h",
        weight: "1.2kg",
      },
      title: {
        values: "Orbit Widget Pro",
      },
    },
    confidence: 0.9,
  });

  assert.deepEqual(result.fields.specs, {
    values: [],
    source: "llm",
    rawObject: JSON.stringify({
      battery: "18h",
      weight: "1.2kg",
    }),
  });
  assert.deepEqual(result.fields.title, {
    values: ["Orbit Widget Pro"],
    source: "llm",
  });
});

test("normalizeExtractResult rejects invalid nested values", () => {
  assert.throws(
    () =>
      normalizeExtractResult({
        fields: {
          title: {
            values: [{ nested: true }],
          },
        },
        confidence: 0.9,
      }),
    /primitive values/,
  );
});

test("normalizeExtractResult clamps confidence into the documented range", () => {
  const high = normalizeExtractResult({ fields: {}, confidence: "1.4" });
  const low = normalizeExtractResult({ fields: {}, confidence: -0.2 });

  assert.equal(high.confidence, 1);
  assert.equal(low.confidence, 0);
});

test("validateTemplateResult accepts JSON-LD-only templates", () => {
  assert.doesNotThrow(() =>
    validateTemplateResult({
      template: {
        name: "article-jsonld",
        jsonld: [
          {
            name: "headline",
            type: "Article",
            path: "headline",
          },
        ],
      },
    }),
  );
});

test("validateTemplateResult accepts regex-only templates", () => {
  assert.doesNotThrow(() =>
    validateTemplateResult({
      template: {
        name: "product-regex",
        regex: [
          {
            name: "sku",
            pattern: "SKU:\\s*([A-Z0-9-]+)",
            source: "html",
            group: 1,
          },
        ],
      },
    }),
  );
});

test("validateTemplateResult accepts mixed extraction strategies", () => {
  assert.doesNotThrow(() =>
    validateTemplateResult({
      template: {
        name: "mixed-template",
        selectors: [
          {
            name: "title",
            selector: "h1",
          },
        ],
        jsonld: [
          {
            name: "price",
            path: "offers.price",
          },
        ],
        regex: [
          {
            name: "sku",
            pattern: "SKU:\\s*([A-Z0-9-]+)",
          },
        ],
      },
    }),
  );
});

test("validateTemplateResult rejects empty templates with no extraction strategies", () => {
  assert.throws(
    () =>
      validateTemplateResult({
        template: {
          name: "empty-template",
        },
      }),
    /at least one selector, jsonld rule, or regex rule/,
  );
});

test("validateTemplateResult rejects selector rules with empty selectors", () => {
  assert.throws(
    () =>
      validateTemplateResult({
        template: {
          name: "selector-template",
          selectors: [
            {
              name: "title",
              selector: "   ",
            },
          ],
        },
      }),
    /selector title must include a selector/,
  );
});

test("validateTemplateResult rejects JSON-LD rules with empty names", () => {
  assert.throws(
    () =>
      validateTemplateResult({
        template: {
          name: "jsonld-template",
          jsonld: [
            {
              name: "  ",
              path: "headline",
            },
          ],
        },
      }),
    /jsonld rule at index 0 must include a name/,
  );
});

test("validateTemplateResult rejects regex rules with empty patterns", () => {
  assert.throws(
    () =>
      validateTemplateResult({
        template: {
          name: "regex-template",
          regex: [
            {
              name: "sku",
              pattern: "   ",
            },
          ],
        },
      }),
    /regex sku must include a pattern/,
  );
});

test("validateTemplateResult rejects regex rules with unsupported sources", () => {
  assert.throws(
    () =>
      validateTemplateResult({
        template: {
          name: "regex-template",
          regex: [
            {
              name: "sku",
              pattern: "SKU:\\s*(.+)",
              source: "headers",
            },
          ],
        },
      }),
    /regex sku source must be text, html, or url/,
  );
});
