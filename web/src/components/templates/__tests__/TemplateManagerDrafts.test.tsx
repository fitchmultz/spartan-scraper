/**
 * Purpose: Verify TemplateManager draft persistence and replacement safeguards.
 * Responsibilities: Cover draft resume/discard flows, dirty-draft replacement confirmation, and save-failure retry behavior.
 * Scope: Local draft lifecycle behavior for TemplateManager only.
 * Usage: Run under Vitest as part of the web test suite.
 * Invariants/Assumptions: Draft state persists through storage-backed remounts and operator-confirmed discard remains explicit.
 */

import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  renderTemplateManager,
  setupTemplateManagerTest,
  getTemplateApiMocks,
} from "./templateManagerTestHarness";

setupTemplateManagerTest();

describe("TemplateManagerDrafts", () => {
  it("restores a closed local template draft after the workspace remounts and lets operators discard it intentionally", async () => {
    const firstRender = await renderTemplateManager();

    fireEvent.click(screen.getByRole("button", { name: /new template/i }));
    fireEvent.change(await screen.findByLabelText(/template name/i), {
      target: { value: "draft-template" },
    });
    fireEvent.change(screen.getByLabelText(/css selector/i), {
      target: { value: "article h1" },
    });

    fireEvent.click(screen.getByRole("button", { name: /^close$/i }));

    expect(
      await screen.findByRole("button", { name: /resume template draft/i }),
    ).toBeInTheDocument();

    firstRender.unmount();

    await renderTemplateManager();

    fireEvent.click(
      await screen.findByRole("button", { name: /resume template draft/i }),
    );

    expect(screen.getByLabelText(/template name/i)).toHaveValue(
      "draft-template",
    );
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /discard draft/i }));

    const confirmDialog = await screen.findByRole("alertdialog");
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /discard draft/i }),
    );

    await waitFor(() => {
      expect(
        screen.queryByRole("button", { name: /resume template draft/i }),
      ).not.toBeInTheDocument();
    });
  });

  it("warns before replacing a dirty local template draft when switching to another saved template", async () => {
    getTemplateApiMocks().getTemplate.mockImplementation(async ({ path }) => ({
      data:
        path.name === "docs"
          ? {
              name: "docs",
              is_built_in: false,
              template: {
                name: "docs",
                selectors: [
                  { name: "title", selector: "main h1", attr: "text" },
                ],
              },
            }
          : {
              name: "news",
              is_built_in: false,
              template: {
                name: "news",
                selectors: [{ name: "title", selector: "h1", attr: "text" }],
              },
            },
      error: undefined as never,
      request: new Request(`http://127.0.0.1:8741/v1/templates/${path.name}`),
      response: new Response(),
    }));

    await renderTemplateManager({ templateNames: ["news", "docs"] });

    expect(await screen.findByLabelText(/template name/i)).toHaveValue("news");

    fireEvent.change(screen.getByLabelText(/css selector/i), {
      target: { value: "article h1" },
    });

    await waitFor(() => {
      expect(screen.getByLabelText(/css selector/i)).toHaveValue("article h1");
    });

    fireEvent.click(screen.getByRole("button", { name: /docs/i }));

    let confirmDialog = await screen.findByRole("alertdialog");
    expect(confirmDialog).toHaveTextContent(
      /replace the current template draft/i,
    );
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /keep draft/i }),
    );

    expect(screen.getByLabelText(/template name/i)).toHaveValue("news");
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /docs/i }));

    confirmDialog = await screen.findByRole("alertdialog");
    fireEvent.click(
      within(confirmDialog).getByRole("button", { name: /discard draft/i }),
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/template name/i)).toHaveValue("docs");
    });
    expect(screen.getByDisplayValue("main h1")).toBeInTheDocument();
  });

  it("keeps the working template draft after a save failure so the operator can retry", async () => {
    getTemplateApiMocks().createTemplate.mockResolvedValue({
      data: undefined,
      error: { error: "template name already exists" },
      request: new Request("http://127.0.0.1:8741/v1/templates"),
      response: new Response(null, { status: 400 }),
    });

    await renderTemplateManager();

    fireEvent.click(screen.getByRole("button", { name: /new template/i }));
    fireEvent.change(await screen.findByLabelText(/template name/i), {
      target: { value: "draft-template" },
    });
    fireEvent.change(screen.getByLabelText(/field name/i), {
      target: { value: "title" },
    });
    fireEvent.change(screen.getByLabelText(/css selector/i), {
      target: { value: "article h1" },
    });

    fireEvent.click(screen.getByRole("button", { name: /save template/i }));

    await waitFor(() => {
      expect(getTemplateApiMocks().createTemplate).toHaveBeenCalledWith(
        expect.objectContaining({
          body: expect.objectContaining({
            name: "draft-template",
            selectors: [
              expect.objectContaining({
                name: "title",
                selector: "article h1",
              }),
            ],
          }),
        }),
      );
    });

    await waitFor(() => {
      expect(screen.getAllByText(/template name already exists/i)).toHaveLength(
        2,
      );
    });
    expect(screen.getByLabelText(/template name/i)).toHaveValue(
      "draft-template",
    );
    expect(screen.getByDisplayValue("article h1")).toBeInTheDocument();
  });
});
