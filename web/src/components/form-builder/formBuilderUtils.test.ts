/**
 * Purpose: Verify form builder utils behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import {
  buildInitialFieldValues,
  buildSelectedFormFieldValues,
  formTypeOptions,
} from "./formBuilderUtils";

describe("formBuilderUtils", () => {
  it("builds blank initial values for all detected fields", () => {
    expect(
      buildInitialFieldValues([
        { allFields: [{ fieldName: "email" }, { fieldName: "password" }] },
        { allFields: [{ fieldName: "query" }] },
      ]),
    ).toEqual({
      email: "",
      password: "",
      query: "",
    });
  });

  it("keeps only filled values for the selected form", () => {
    expect(
      buildSelectedFormFieldValues(
        { allFields: [{ fieldName: "email" }, { fieldName: "password" }] },
        { email: "person@example.com", password: "", ignored: "x" },
      ),
    ).toEqual({
      email: "person@example.com",
    });
  });

  it("exposes stable form type options", () => {
    expect(formTypeOptions[0]).toEqual({ value: "", label: "All Types" });
    expect(formTypeOptions.at(-1)).toEqual({
      value: "survey",
      label: "Survey",
    });
  });
});
