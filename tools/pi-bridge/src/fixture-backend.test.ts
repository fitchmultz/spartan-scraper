import test from "node:test";
import assert from "node:assert/strict";
import { FixtureBackend } from "./fixture-backend.js";
import { CAPABILITY_EXTRACT_NATURAL } from "./protocol.js";
import { validateExtractResult, validateTemplateResult } from "./validation.js";

test("FixtureBackend produces deterministic extract output", () => {
  const backend = new FixtureBackend("fixture", {
    [CAPABILITY_EXTRACT_NATURAL]: ["openai/gpt-5.4"],
  });

  const result = backend.extract(CAPABILITY_EXTRACT_NATURAL, {
    html: "<html></html>",
    url: "https://example.com",
    mode: "natural_language",
    fields: ["title"],
  });

  assert.equal(result.fields.title?.values?.[0], "fixture:title");
  assert.equal(result.provider, "fixture");
});

test("validation helpers reject malformed outputs", () => {
  assert.throws(() =>
    validateExtractResult({ fields: {}, confidence: Number.NaN }),
  );
  assert.throws(() =>
    validateTemplateResult({
      template: { name: "", selectors: [] },
    }),
  );
});
