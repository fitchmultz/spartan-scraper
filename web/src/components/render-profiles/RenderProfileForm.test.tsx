/**
 * Purpose: Verify render-profile draft editing clears submit errors when the user changes a field.
 * Responsibilities: Exercise the shared draft form chrome, submit failure handling, and field-driven error reset behavior.
 * Scope: RenderProfileForm component only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Editing any draft field should clear prior submit errors without waiting for a rerender side effect.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  createEmptyRenderProfileInput,
  RenderProfileForm,
} from "./RenderProfileForm";

describe("RenderProfileForm", () => {
  it("clears submit errors as soon as a draft field changes", () => {
    const initialValue = {
      ...createEmptyRenderProfileInput(),
      name: "pricing-profile",
      hostPatterns: ["example.com"],
    };

    const onSubmit = vi.fn(() => {
      throw new Error("Save failed");
    });

    render(
      <RenderProfileForm
        initialValue={initialValue}
        onSubmit={onSubmit}
        onCancel={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /create/i }));

    expect(screen.getByRole("alert")).toHaveTextContent("Save failed");

    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "example.com, example.org" },
    });

    expect(screen.queryByRole("alert")).toBeNull();
  });
});
