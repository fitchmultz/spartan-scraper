import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { RenderProfileEditor } from "./RenderProfileEditor";
import * as api from "../../api";

vi.mock("../../api", () => ({
  getV1RenderProfiles: vi.fn(),
  postV1RenderProfiles: vi.fn(),
  putV1RenderProfilesByName: vi.fn(),
  deleteV1RenderProfilesByName: vi.fn(),
}));

describe("RenderProfileEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getV1RenderProfiles).mockResolvedValue({
      data: { profiles: [] },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(),
    });
  });

  it("keeps the create form open and shows the API error when create fails", async () => {
    vi.mocked(api.postV1RenderProfiles).mockResolvedValue({
      data: undefined,
      error: { error: "invalid hostPatterns" },
      request: new Request("http://localhost:8741/v1/render-profiles"),
      response: new Response(null, { status: 400 }),
    });

    const onError = vi.fn();
    render(<RenderProfileEditor onError={onError} />);

    await waitFor(() => {
      expect(api.getV1RenderProfiles).toHaveBeenCalledTimes(1);
    });

    fireEvent.click(screen.getByRole("button", { name: /create profile/i }));
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "news" },
    });
    fireEvent.change(screen.getByLabelText(/host patterns/i), {
      target: { value: "bad pattern" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^create$/i }));

    await waitFor(() => {
      expect(api.postV1RenderProfiles).toHaveBeenCalledWith({
        body: expect.objectContaining({
          name: "news",
          hostPatterns: ["bad pattern"],
        }),
      });
    });

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "invalid hostPatterns",
    );
    expect(
      screen.getByRole("heading", { name: /create new profile/i }),
    ).toBeInTheDocument();
    expect(api.getV1RenderProfiles).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith("invalid hostPatterns");
  });
});
