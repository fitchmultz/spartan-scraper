/**
 * Purpose: Verify template workspace draft codecs and payload validation stay aligned with the split route helpers.
 * Responsibilities: Cover draft formatting, snapshot normalization, and save-payload validation for template JSON blocks.
 * Scope: Template workspace helper behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 */

import { describe, expect, it } from "vitest";

import type { SelectorRule, Template } from "../../api";
import {
  buildDraftFromTemplate,
  buildTemplateSnapshot,
  type SelectorDraft,
  type TemplateDraftState,
} from "./templateRouteControllerShared";
import { buildTemplatePayload } from "./useTemplateMutationActions";

const selectorRule: SelectorRule = {
  name: " title ",
  selector: " article h1 ",
  attr: " text ",
  trim: true,
  all: false,
  required: false,
  join: "  ",
};

const selectorDraft: SelectorDraft = {
  id: "selector-1",
  rule: selectorRule,
};

const draft: TemplateDraftState = {
  name: "  article  ",
  selectors: [selectorDraft],
  jsonldText: "",
  regexText: "",
  normalizeText: "",
};

describe("templateWorkspaceHelpers", () => {
  it("formats template JSON blocks when seeding a draft", () => {
    const template: Template = {
      name: "article",
      selectors: [selectorRule],
      jsonld: [{ name: "author", type: "Article", path: "author.name" }],
      regex: [
        {
          name: "price",
          pattern: "\\$([0-9.]+)",
          group: 1,
          source: "text",
        },
      ],
      normalize: { titleField: "title" },
    };

    const seededDraft = buildDraftFromTemplate(template);

    expect(seededDraft.jsonldText).toBe(
      JSON.stringify(template.jsonld, null, 2),
    );
    expect(seededDraft.regexText).toBe(JSON.stringify(template.regex, null, 2));
    expect(seededDraft.normalizeText).toBe(
      JSON.stringify(template.normalize, null, 2),
    );
  });

  it("builds normalized snapshots without surfacing invalid advanced JSON", () => {
    const snapshot = buildTemplateSnapshot({
      ...draft,
      jsonldText: "{bad json}",
      regexText:
        '[{"name":"price","pattern":"\\\\$([0-9.]+)","group":1,"source":"text"}]',
      normalizeText: "false",
    });

    expect(snapshot).toEqual({
      name: "article",
      selectors: [
        {
          name: "title",
          selector: "article h1",
          attr: "text",
          trim: true,
          all: false,
          required: false,
          join: undefined,
        },
      ],
      regex: [
        {
          name: "price",
          pattern: "\\$([0-9.]+)",
          group: 1,
          source: "text",
        },
      ],
    });
    expect(snapshot.jsonld).toBeUndefined();
    expect(snapshot.normalize).toBeUndefined();
  });

  it("builds save payloads from valid draft data and rejects malformed JSON blocks", () => {
    expect(buildTemplatePayload(draft)).toEqual({
      payload: {
        name: "article",
        selectors: [
          {
            name: "title",
            selector: "article h1",
            attr: "text",
            trim: true,
            all: false,
            required: false,
            join: undefined,
          },
        ],
      },
    });

    expect(
      buildTemplatePayload({
        ...draft,
        jsonldText: '{"name":"author"}',
      }),
    ).toEqual({
      error: "JSON-LD rules must be a JSON array.",
    });

    expect(
      buildTemplatePayload({
        ...draft,
        normalizeText: "[]",
      }),
    ).toEqual({
      error: "Normalization settings must be a JSON object.",
    });
  });
});
