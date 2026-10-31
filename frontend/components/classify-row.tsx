"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { api, type Transaction } from "@/lib/api";
import { typeForCategory } from "@/lib/classify";
import { displayPayeeName } from "@/lib/payee";
import {
  ATTENTION_ACCENT,
  ATTENTION_CHIP_STYLE,
  CategoryIcon,
  categoryAccent,
  categoryChipStyle,
} from "@/components/category-icon";
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
  if (t.category === "Other") return "Uncategorized";
  return t.classification_status;
}

export function displayCategory(cat?: string | null): string {
  if (!cat || cat === "Other") return "Uncategorized";
  return cat;
}

/** Label for “apply to all” — same payee extraction as the Name column (not rail tokens). */
function payeeHint(txn: Transaction): string {
  const name = displayPayeeName(txn.merchant, txn.original_description).trim();
  if (!name || name === "—") return "";
  // Skip if we still only got rails (stale merchant / weak extract).
  if (/^(WDL|TFR|DEP|UPI|IMPS|NEFT)(\s|$)/i.test(name)) return "";
  return name.length > 28 ? `${name.slice(0, 26)}…` : name;
}

/** Category chip: same shape for all rows; amber help icon only when attention needed. */
export function CategoryCell({
  txn,
  categories,
  onSaved,
}: {
  txn: Transaction;
  categories: string[];
  onSaved: (updated: Transaction, meta?: { applied_count: number; pattern?: string }) => void;
}) {
  const ambiguous = needsAttention(txn);
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState(
    txn.category && txn.category !== "Other" ? txn.category : "",
  );
  const [saving, setSaving] = useState<"one" | "all" | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const hint = useMemo(
    () => payeeHint(txn),
    [txn.id, txn.merchant, txn.original_description],
  );

  useEffect(() => {
    setCategory(txn.category && txn.category !== "Other" ? txn.category : "");
  }, [txn.id, txn.category]);

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

  async function save(mode: "one" | "all") {
    if (!category) return;
    setSaving(mode);
    setErr(null);
    try {
      const res = await api.correct(txn.id, {
        category,
        create_rule: mode === "all",
        apply_to_matching: mode === "all",
        type: typeForCategory(category, txn.direction),
      });
      setOpen(false);
      onSaved(res.transaction, {
        applied_count: res.applied_count,
        pattern: res.pattern,
      });
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(null);
    }
  }

  const label = displayCategory(txn.category);
  const title = ambiguous
    ? `${statusLabel(txn)} — click to categorize`
    : `Change category (${label}${txn.subcategory ? ` / ${txn.subcategory}` : ""})`;
  const tint = ambiguous ? ATTENTION_CHIP_STYLE : categoryChipStyle(txn.category);
  const accent = ambiguous ? ATTENTION_ACCENT : categoryAccent(txn.category);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        title={title}
        onClick={() => setOpen((v) => !v)}
        style={tint}
        className={cn(
          "inline-flex max-w-full items-center gap-1.5 rounded-md border px-2 py-1 text-left text-xs transition-opacity hover:opacity-90",
          open && "ring-1 ring-foreground/20",
        )}
      >
        <CategoryIcon category={txn.category} style={{ color: accent }} />
        <span className="truncate text-muted-foreground">{label}</span>
      </button>

      {open && (
        <div className="absolute left-0 top-full z-30 mt-1.5 w-64 rounded-md border border-border bg-card p-2 shadow-lg">
          <p className="mb-2 px-1 text-[11px] text-muted-foreground">
            {ambiguous ? statusLabel(txn) : "Recategorize"}
          </p>
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
                {c === "Other" ? "Uncategorized" : c}
              </option>
            ))}
          </select>
          <div className="mt-2 flex flex-col gap-1.5">
            <Button
              type="button"
              size="sm"
              className="h-8 w-full text-xs"
              disabled={!category || saving !== null}
              onClick={() => save("one")}
            >
              {saving === "one" ? "…" : "This only"}
            </Button>
            {hint ? (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                className="h-8 w-full text-xs"
                disabled={!category || saving !== null}
                onClick={() => save("all")}
              >
                {saving === "all" ? "…" : `All “${hint}”`}
              </Button>
            ) : null}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
          </div>
          {err && <p className="mt-1 px-1 text-[11px] text-red-300">{err}</p>}
        </div>
      )}
    </div>
  );
}
