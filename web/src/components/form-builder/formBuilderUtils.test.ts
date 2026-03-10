/**
 * formBuilderUtils.test
 *
 * Purpose:
 * - Verify the form-builder helper functions keep field mapping behavior stable.
 *
 * Responsibilities:
 * - Cover initial field-value generation.
 * - Confirm selected-form submissions exclude empty or unrelated values.
 * - Lock in the shared form-type option list.
 *
 * Scope:
 * - Unit tests for pure form-builder helpers only.
 *
 * Usage:
 * - Run via Vitest as part of frontend validation.
 *
 * Invariants/Assumptions:
 * - Field names are the stable keys used by the form builder UI.
 * - Shared options are intentionally ordered for display.
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
