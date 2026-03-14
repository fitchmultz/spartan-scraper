import { useId, useRef, useState, type ClipboardEvent } from "react";

import {
  ACCEPTED_AI_IMAGE_MIME_TYPES,
  aiImageDataURL,
  extractClipboardImageFiles,
  readAIImageFiles,
  type AttachedAIImage,
} from "../lib/ai-image-utils";

interface AIImageAttachmentsProps {
  images: AttachedAIImage[];
  onChange: (images: AttachedAIImage[]) => void;
  disabled?: boolean;
}

const ACCEPT = ACCEPTED_AI_IMAGE_MIME_TYPES.join(",");

export function AIImageAttachments({
  images,
  onChange,
  disabled = false,
}: AIImageAttachmentsProps) {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleFiles = async (files: File[]) => {
    if (files.length === 0 || disabled) {
      return;
    }
    try {
      const next = await readAIImageFiles(files, images);
      onChange([...images, ...next]);
      setError(null);
      if (inputRef.current) {
        inputRef.current.value = "";
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add images.");
    }
  };

  const handlePaste = async (event: ClipboardEvent<HTMLDivElement>) => {
    const files = extractClipboardImageFiles(event.clipboardData.items);
    if (files.length === 0) {
      return;
    }
    event.preventDefault();
    await handleFiles(files);
  };

  return (
    <div className="rounded-md border border-slate-700 bg-slate-900/60 p-4">
      <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-medium text-slate-200">
            Reference Images (optional)
          </h3>
          <p className="mt-1 text-sm text-slate-400">
            Upload or paste screenshots and other page images as bounded visual
            context. These images are used only for this request and are not
            stored as job artifacts.
          </p>
        </div>
        {images.length > 0 ? (
          <button
            type="button"
            className="btn btn--secondary"
            onClick={() => {
              onChange([]);
              setError(null);
              if (inputRef.current) {
                inputRef.current.value = "";
              }
            }}
            disabled={disabled}
          >
            Clear Images
          </button>
        ) : null}
      </div>

      <div className="flex flex-wrap gap-3">
        <label
          htmlFor={inputId}
          className={`inline-flex cursor-pointer items-center gap-2 rounded-md border border-slate-600 px-3 py-2 text-sm font-medium transition-colors ${
            disabled
              ? "cursor-not-allowed border-slate-800 text-slate-600"
              : "text-slate-200 hover:border-slate-500 hover:bg-slate-800"
          }`}
        >
          <span>Upload Images</span>
          <input
            ref={inputRef}
            id={inputId}
            type="file"
            accept={ACCEPT}
            multiple
            className="sr-only"
            disabled={disabled}
            onChange={(event) => {
              void handleFiles(Array.from(event.target.files ?? []));
            }}
          />
        </label>
        <div
          className={`min-w-[16rem] flex-1 rounded-md border border-dashed px-3 py-2 text-sm ${
            disabled
              ? "border-slate-800 text-slate-600"
              : "border-slate-600 text-slate-300"
          }`}
          onPaste={(event) => {
            void handlePaste(event);
          }}
          tabIndex={disabled ? -1 : 0}
        >
          Paste an image here with ⌘/Ctrl+V.
        </div>
      </div>

      <p className="mt-2 text-xs text-slate-500">
        Up to 4 images total. PNG, JPEG, WebP, and GIF only. 1 MiB per image, 4
        MiB combined.
      </p>

      {error ? (
        <div className="mt-3 rounded-md border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-sm text-rose-100">
          {error}
        </div>
      ) : null}

      {images.length > 0 ? (
        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {images.map((image) => (
            <div
              key={image.id}
              className="rounded-md border border-slate-700 bg-slate-950/70 p-2"
            >
              <img
                src={aiImageDataURL(image)}
                alt={image.name}
                className="mb-2 h-32 w-full rounded object-cover"
              />
              <div className="space-y-1 text-xs text-slate-400">
                <div className="truncate font-medium text-slate-200">
                  {image.name}
                </div>
                <div>{image.mimeType}</div>
                <div>{Math.max(1, Math.round(image.size / 1024))} KB</div>
              </div>
              <button
                type="button"
                className="btn btn--secondary mt-3 w-full"
                onClick={() => {
                  onChange(
                    images.filter((candidate) => candidate.id !== image.id),
                  );
                  setError(null);
                }}
                disabled={disabled}
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
