/**
 * Purpose: Verify the export-schedule editor keeps optional AI helpers non-blocking and coherent.
 * Responsibilities: Assert manual transform/shape editing remains available, AI launchers open when available, and AI actions disable cleanly when AI is off.
 * Scope: ExportScheduleForm behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Export schedules must remain fully configurable without AI assistance.
 */

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { defaultFormData } from "../../lib/export-schedule-utils";
import { ExportScheduleForm } from "./ExportScheduleForm";

describe("ExportScheduleForm", () => {
  it("shows transform controls and AI helper when no shape is configured", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <ExportScheduleForm
        formData={{
          ...defaultFormData,
          transformExpression: "{title: title, url: url}",
        }}
        formError={null}
        formSubmitting={false}
        isEditing={false}
        onChange={onChange}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(screen.getByText(/Result Transform/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /AI Suggest Transform/i }),
    ).toBeEnabled();
    expect(
      screen.getByDisplayValue(/\{title: title, url: url\}/i),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Clear Transform/i }));

    expect(onChange).toHaveBeenCalledWith({
      transformExpression: "",
      transformLanguage: "jmespath",
    });
  });

  it("shows export shaping controls for supported formats", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <ExportScheduleForm
        formData={{
          ...defaultFormData,
          format: "md",
          localPath: "exports/{job_id}.md",
          pathTemplate: "exports/{job_id}.md",
        }}
        formError={null}
        formSubmitting={false}
        isEditing={false}
        onChange={onChange}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(screen.getByText(/Export Shaping/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /AI Suggest Shape/i }),
    ).toBeEnabled();

    await user.type(screen.getByLabelText(/Top-level fields/i), "url");

    expect(onChange).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /AI Suggest Shape/i }));

    expect(
      await screen.findByRole("heading", { name: /Shape Export with AI/i }),
    ).toBeInTheDocument();
  });

  it("disables AI suggestion actions and explains the manual path when AI is unavailable", () => {
    render(
      <ExportScheduleForm
        formData={{
          ...defaultFormData,
          format: "md",
          localPath: "exports/{job_id}.md",
          pathTemplate: "exports/{job_id}.md",
        }}
        formError={null}
        formSubmitting={false}
        isEditing={false}
        onChange={vi.fn()}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
        aiStatus={{
          status: "disabled",
          message: "AI helpers are optional and currently disabled.",
        }}
      />,
    );

    expect(
      screen.getByText(
        /configure transforms and shapes manually in this form/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /AI Suggest Transform/i }),
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: /AI Suggest Shape/i }),
    ).toBeDisabled();
    expect(screen.getByLabelText(/Transform language/i)).toBeEnabled();
    expect(screen.getByLabelText(/Top-level fields/i)).toBeEnabled();
  });

  it("locks transform controls when shape configuration is staged", () => {
    render(
      <ExportScheduleForm
        formData={{
          ...defaultFormData,
          format: "md",
          localPath: "exports/{job_id}.md",
          pathTemplate: "exports/{job_id}.md",
          shapeTopLevelFields: "url",
        }}
        formError={null}
        formSubmitting={false}
        isEditing={false}
        onChange={vi.fn()}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(
      screen.getByText(/clear the shape before configuring a saved transform/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /AI Suggest Transform/i }),
    ).toBeDisabled();
  });

  it("shows unsupported-format guidance and allows clearing staged shape fields", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <ExportScheduleForm
        formData={{
          ...defaultFormData,
          format: "json",
          shapeTopLevelFields: "url",
          shapeFieldLabels: "title=Page Title",
        }}
        formError={null}
        formSubmitting={false}
        isEditing
        onChange={onChange}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
    );

    expect(
      screen.getByText(
        /JSON and JSON Lines exports always ship the full structured payload/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Shape fields are currently staged in the form/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /AI Suggest Shape/i }),
    ).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /Clear Shape/i }));

    expect(onChange).toHaveBeenCalledWith({
      shapeTopLevelFields: "",
      shapeNormalizedFields: "",
      shapeEvidenceFields: "",
      shapeSummaryFields: "",
      shapeFieldLabels: "",
      shapeEmptyValue: "",
      shapeMultiValueJoin: "",
      shapeMarkdownTitle: "",
    });
  });
});
