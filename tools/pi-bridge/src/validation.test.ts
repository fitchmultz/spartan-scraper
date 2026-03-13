import assert from "node:assert/strict";
import test from "node:test";
import { normalizeExtractResult } from "./validation.js";

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
