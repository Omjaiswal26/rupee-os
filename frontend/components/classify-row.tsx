"use client";

import { useEffect, useRef, useState } from "react";
import { CircleHelp } from "lucide-react";
import { api, type Transaction } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function needsAttention(t: Transaction): boolean {
  const st = t.classification_status;
  if (st === "pending" || st === "needs_review") return true;
  if (!t.category || t.category === "Other") return true;
  return false;
}

export function statusLabel(t: Transaction): string {
  if (t.classification_status === "pending") return "Pending classification";
  if (t.classification_status === "needs_review") return "Needs review";
  if (!t.category) return "Uncategorized";
  if (t.category === "Other") return "Ambiguous (Other)";
  return t.classification_status;
}

/** Category cell: plain text, or a small badge that opens a categorize popover. */
export function CategoryCell({
  txn,
  categories,
  onSaved,
}: {
  txn: Transaction;
  categories: string[];
  onSaved: (updated: Transaction) => void;
}) {
  const ambiguous = needsAttention(txn);
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState(txn.category && txn.category !== "Other" ? txn.category : "");
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  async function save() {
    if (!category) return;
    setSaving(true);
    setErr(null);
    try {
      const res = await api.correct(txn.id, {
        category,
        create_rule: true,
        type: txn.direction === "credit" ? "income" : "expense",
      });
      setOpen(false);
      onSaved(res.transaction);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  if (!ambiguous) {
    return (
      <span className="text-muted-foreground">
        {txn.category || "—"}
        {txn.subcategory ? <span className="text-xs"> / {txn.subcategory}</span> : null}
      </span>
    );
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        title={statusLabel(txn) + " — click to categorize"}
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "inline-flex items-center gap-1.5 rounded-md border border-border/80 px-2 py-1 text-xs text-muted-foreground transition-colors hover:border-amber-500/50 hover:text-foreground",
          open && "border-amber-500/50 text-foreground",
        )}
      >
        <CircleHelp className="h-3.5 w-3.5 text-amber-400/90" strokeWidth={2} />
        <span>{txn.category && txn.category !== "Other" ? txn.category : "Uncategorized"}</span>
      </button>

      {open && (
        <div className="absolute left-0 top-full z-30 mt-1.5 w-56 rounded-md border border-border bg-card p-2 shadow-lg">
          <p className="mb-2 px-1 text-[11px] text-muted-foreground">{statusLabel(txn)}</p>
          <select
            autoFocus
            className="h-8 w-full rounded-md border border-input bg-transparent px-2 text-xs"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            <option value="" disabled>
              Choose category
            </option>
            {categories.map((c) => (
              <option key={c} value={c} className="bg-card">
                {c}
              </option>
            ))}
          </select>
          <div className="mt-2 flex justify-end gap-1.5">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              size="sm"
              className="h-7 px-2 text-xs"
              disabled={!category || saving}
              onClick={save}
            >
              {saving ? "…" : "Save"}
            </Button>
          </div>
          {err && <p className="mt-1 px-1 text-[11px] text-red-300">{err}</p>}
        </div>
      )}
    </div>
  );
}
