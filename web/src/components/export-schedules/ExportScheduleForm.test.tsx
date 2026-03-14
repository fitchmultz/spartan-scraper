import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { defaultFormData } from "../../lib/export-schedule-utils";
import { ExportScheduleForm } from "./ExportScheduleForm";

describe("ExportScheduleForm", () => {
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
