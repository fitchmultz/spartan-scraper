import { describe, expect, it } from "vitest";

import {
  MAX_AI_IMAGE_BYTES,
  MAX_AI_IMAGES,
  aiImageDataURL,
  extractClipboardImageFiles,
  readAIImageFiles,
  toAIImagePayloads,
  type AttachedAIImage,
} from "./ai-image-utils";

describe("ai-image-utils", () => {
  it("maps attached images to API payloads", () => {
    const images: AttachedAIImage[] = [
      {
        id: "img-1",
        name: "ref.png",
        mimeType: "image/png",
        size: 4,
        data: "ZmFrZQ==",
      },
    ];

    expect(toAIImagePayloads(images)).toEqual([
      { data: "ZmFrZQ==", mime_type: "image/png" },
    ]);
    expect(toAIImagePayloads([])).toBeUndefined();
  });

  it("builds preview data URLs", () => {
    expect(
      aiImageDataURL({
        id: "img-1",
        name: "ref.png",
        mimeType: "image/png",
        size: 4,
        data: "ZmFrZQ==",
      }),
    ).toBe("data:image/png;base64,ZmFrZQ==");
  });

  it("extracts image files from clipboard items", () => {
    const image = new File(["fake"], "ref.png", { type: "image/png" });
    const text = new File(["note"], "note.txt", { type: "text/plain" });
    const items = [
      { kind: "string", getAsFile: () => null },
      { kind: "file", getAsFile: () => text },
      { kind: "file", getAsFile: () => image },
    ] as unknown as DataTransferItemList;

    expect(extractClipboardImageFiles(items)).toEqual([image]);
  });

  it("reads image files into attached image objects", async () => {
    const file = new File(["fake"], "ref.png", { type: "image/png" });

    const images = await readAIImageFiles([file]);

    expect(images).toHaveLength(1);
    expect(images[0]).toMatchObject({
      name: "ref.png",
      mimeType: "image/png",
      size: 4,
      data: "ZmFrZQ==",
    });
    expect(images[0].id).toBeTruthy();
  });

  it("rejects too many attachments before reading files", async () => {
    const files = Array.from(
      { length: MAX_AI_IMAGES + 1 },
      (_, index) =>
        new File([`img-${index}`], `ref-${index}.png`, { type: "image/png" }),
    );

    await expect(readAIImageFiles(files)).rejects.toThrow(
      `You can attach at most ${MAX_AI_IMAGES} images.`,
    );
  });

  it("rejects unsupported image mime types", async () => {
    const file = new File(["fake"], "ref.svg", { type: "image/svg+xml" });

    await expect(readAIImageFiles([file])).rejects.toThrow(
      "ref.svg must be PNG, JPEG, WebP, or GIF.",
    );
  });

  it("rejects oversized images", async () => {
    const file = new File([new Uint8Array(MAX_AI_IMAGE_BYTES + 1)], "big.png", {
      type: "image/png",
    });

    await expect(readAIImageFiles([file])).rejects.toThrow(
      "big.png exceeds the 1 MiB per-image limit.",
    );
  });
});
