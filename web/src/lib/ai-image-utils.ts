/**
 * Purpose: Provide reusable ai image utils helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
 */

import type { AiImageInput } from "../api";

export const MAX_AI_IMAGES = 4;
export const MAX_AI_IMAGE_BYTES = 1024 * 1024;
export const MAX_TOTAL_AI_IMAGE_BYTES = 4 * 1024 * 1024;

type AcceptedAIImageMimeType = AiImageInput["mime_type"];

export const ACCEPTED_AI_IMAGE_MIME_TYPES = [
  "image/png",
  "image/jpeg",
  "image/webp",
  "image/gif",
] as const satisfies readonly AcceptedAIImageMimeType[];

export interface AttachedAIImage {
  id: string;
  name: string;
  mimeType: AcceptedAIImageMimeType;
  size: number;
  data: string;
}

export type AIImageInputPayload = AiImageInput;

export async function readAIImageFiles(
  files: File[],
  existing: AttachedAIImage[] = [],
): Promise<AttachedAIImage[]> {
  if (files.length === 0) {
    return [];
  }
  validateAIImageBudget(existing, files);
  return Promise.all(files.map(readAIImageFile));
}

export function extractClipboardImageFiles(
  items: DataTransferItemList,
): File[] {
  const files: File[] = [];
  for (const item of Array.from(items)) {
    if (item.kind !== "file") {
      continue;
    }
    const file = item.getAsFile();
    if (!file?.type.startsWith("image/")) {
      continue;
    }
    files.push(file);
  }
  return files;
}

export function toAIImagePayloads(
  images: AttachedAIImage[],
): AIImageInputPayload[] | undefined {
  if (images.length === 0) {
    return undefined;
  }
  return images.map(({ data, mimeType }) => ({
    data,
    mime_type: mimeType,
  }));
}

export function aiImageDataURL(image: AttachedAIImage): string {
  return `data:${image.mimeType};base64,${image.data}`;
}

function isAcceptedAIImageMimeType(
  value: string,
): value is AcceptedAIImageMimeType {
  return (ACCEPTED_AI_IMAGE_MIME_TYPES as readonly string[]).includes(value);
}

function validateAIImageBudget(existing: AttachedAIImage[], files: File[]) {
  const totalCount = existing.length + files.length;
  if (totalCount > MAX_AI_IMAGES) {
    throw new Error(`You can attach at most ${MAX_AI_IMAGES} images.`);
  }

  let totalBytes = existing.reduce((sum, image) => sum + image.size, 0);
  for (const file of files) {
    if (!isAcceptedAIImageMimeType(file.type)) {
      throw new Error(
        `${file.name || "Pasted image"} must be PNG, JPEG, WebP, or GIF.`,
      );
    }
    if (file.size <= 0) {
      throw new Error(`${file.name || "Pasted image"} is empty.`);
    }
    if (file.size > MAX_AI_IMAGE_BYTES) {
      throw new Error(
        `${file.name || "Pasted image"} exceeds the 1 MiB per-image limit.`,
      );
    }
    totalBytes += file.size;
  }
  if (totalBytes > MAX_TOTAL_AI_IMAGE_BYTES) {
    throw new Error("Attached images exceed the 4 MiB total size limit.");
  }
}

async function readAIImageFile(file: File): Promise<AttachedAIImage> {
  const dataUrl = await readFileAsDataURL(file);
  const parts = dataUrl.split(",", 2);
  const data = parts.length === 2 ? parts[1] : "";
  if (!data) {
    throw new Error(`Failed to read ${file.name || "image"}.`);
  }
  if (!isAcceptedAIImageMimeType(file.type)) {
    throw new Error(
      `${file.name || "Pasted image"} must be PNG, JPEG, WebP, or GIF.`,
    );
  }
  return {
    id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    name: file.name || "Pasted image",
    mimeType: file.type,
    size: file.size,
    data,
  };
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => {
      reject(new Error(`Failed to read ${file.name || "image"}.`));
    };
    reader.onload = () => {
      if (typeof reader.result !== "string") {
        reject(new Error(`Failed to read ${file.name || "image"}.`));
        return;
      }
      resolve(reader.result);
    };
    reader.readAsDataURL(file);
  });
}
