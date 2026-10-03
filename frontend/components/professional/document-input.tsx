"use client";

import { useEffect, useId, useMemo, useState } from "react";
import { FileText, Loader2, Upload, X } from "lucide-react";

import { prepareUpload, UploadError } from "@/lib/prepare-upload";
import { cn } from "@/lib/utils";

type Props = {
  label: string;
  hint?: string;
  value: File | null;
  onChange: (file: File | null) => void;
  required?: boolean;
  allowPdf?: boolean;
};

/** A document picker with a preview. Photos are shrunk before upload. */
export function DocumentInput({ label, hint, value, onChange, required, allowPdf = true }: Props) {
  const id = useId();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const preview = useMemo(
    () => (value && value.type.startsWith("image/") ? URL.createObjectURL(value) : null),
    [value],
  );
  useEffect(
    () => () => {
      if (preview) URL.revokeObjectURL(preview);
    },
    [preview],
  );

  const pick = async (file: File | undefined) => {
    if (!file) return;
    setBusy(true);
    setError(null);
    try {
      onChange(await prepareUpload(file, { allowPdf }));
    } catch (e) {
      onChange(null);
      setError(e instanceof UploadError ? e.message : "Couldn't use this file.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium">
        {label}
        {!required && <span className="font-normal text-muted-foreground"> (optional)</span>}
      </label>
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
      <div
        className={cn(
          "flex items-center gap-3 rounded-lg border border-dashed p-3",
          value && "border-solid",
          error && "border-destructive",
        )}
      >
        {value ? (
          preview ? (
            // A local blob: URL of the chosen photo; next/image can't help here.
            // eslint-disable-next-line @next/next/no-img-element
            <img src={preview} alt="" className="size-14 rounded-md object-cover" />
          ) : (
            <FileText aria-hidden className="size-10 text-muted-foreground" />
          )
        ) : busy ? (
          <Loader2 aria-hidden className="size-6 animate-spin text-muted-foreground" />
        ) : (
          <Upload aria-hidden className="size-6 text-muted-foreground" />
        )}
        <div className="min-w-0 flex-1 text-sm">
          {value ? (
            <span className="block truncate">{value.name}</span>
          ) : (
            <span className="block text-muted-foreground">
              {allowPdf ? "Photo or PDF, up to 5 MB" : "Photo, up to 5 MB"}
            </span>
          )}
          {/* The native input is visually hidden: after a pick it is
              cleared (so the same file can be chosen again) and would
              otherwise say "No file chosen" next to the preview. */}
          <input
            id={id}
            type="file"
            accept={allowPdf ? "image/*,application/pdf" : "image/*"}
            className="peer sr-only"
            onChange={(event) => {
              void pick(event.target.files?.[0]);
              event.target.value = "";
            }}
          />
          <label
            htmlFor={id}
            className="mt-1.5 inline-flex min-h-11 cursor-pointer items-center rounded-[10px] border bg-card px-3.5 py-1.5 sm:min-h-0 text-sm font-medium hover:bg-accent peer-focus-visible:ring-2 peer-focus-visible:ring-ring"
          >
            {value ? "Replace" : "Choose file"}
          </label>
        </div>
        {value && (
          <button
            type="button"
            onClick={() => onChange(null)}
            aria-label={`Remove ${label}`}
            className="rounded-md p-1 hover:bg-accent"
          >
            <X aria-hidden className="size-4" />
          </button>
        )}
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
