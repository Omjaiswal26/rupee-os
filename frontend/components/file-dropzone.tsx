"use client";

import { useCallback, useId, useRef, useState } from "react";
import { FileSpreadsheet, Upload, X } from "lucide-react";
import { cn } from "@/lib/utils";

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

/** Drag-and-drop file picker for bank statements (CSV / XLSX). */
export function FileDropzone({
  file,
  onFile,
  accept = ".csv,.xlsx,.xls",
  disabled = false,
}: {
  file: File | null;
  onFile: (f: File | null) => void;
  accept?: string;
  disabled?: boolean;
}) {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);

  const pick = useCallback(
    (list: FileList | null) => {
      if (!list?.length || disabled) return;
      const f = list[0];
      const name = f.name.toLowerCase();
      if (!name.endsWith(".csv") && !name.endsWith(".xlsx") && !name.endsWith(".xls")) {
        return;
      }
      onFile(f);
    },
    [disabled, onFile],
  );

  return (
    <div className="space-y-2">
      <input
        ref={inputRef}
        id={inputId}
        type="file"
        accept={accept}
        className="sr-only"
        disabled={disabled}
        onChange={(e) => {
          pick(e.target.files);
          e.target.value = "";
        }}
      />

      {!file ? (
        <label
          htmlFor={inputId}
          onDragEnter={(e) => {
            e.preventDefault();
            if (!disabled) setDragging(true);
          }}
          onDragOver={(e) => {
            e.preventDefault();
            if (!disabled) setDragging(true);
          }}
          onDragLeave={(e) => {
            e.preventDefault();
            setDragging(false);
          }}
          onDrop={(e) => {
            e.preventDefault();
            setDragging(false);
            pick(e.dataTransfer.files);
          }}
          className={cn(
            "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center transition-colors",
            dragging
              ? "border-primary bg-primary/10"
              : "border-border bg-muted/20 hover:border-muted-foreground/40 hover:bg-muted/30",
            disabled && "pointer-events-none opacity-50",
          )}
        >
          <Upload className="h-6 w-6 text-muted-foreground" strokeWidth={1.75} />
          <div>
            <p className="text-sm text-foreground">
              Drop statement here, or <span className="font-medium text-primary">browse</span>
            </p>
            <p className="mt-1 text-xs text-muted-foreground">CSV or XLSX · one calendar month</p>
          </div>
        </label>
      ) : (
        <div className="flex items-center gap-3 rounded-lg border border-border bg-muted/20 px-3 py-3">
          <FileSpreadsheet className="h-8 w-8 shrink-0 text-emerald-400/90" strokeWidth={1.5} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-foreground">{file.name}</p>
            <p className="text-xs text-muted-foreground">{formatBytes(file.size)}</p>
          </div>
          <button
            type="button"
            disabled={disabled}
            onClick={() => onFile(null)}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
            aria-label="Remove file"
          >
            <X className="h-4 w-4" />
          </button>
          <button
            type="button"
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
            className="shrink-0 text-xs font-medium text-primary hover:underline"
          >
            Change
          </button>
        </div>
      )}
    </div>
  );
}
