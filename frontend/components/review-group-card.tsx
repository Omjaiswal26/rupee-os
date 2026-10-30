"use client";

import { useMemo, useState } from "react";
import { api, type Transaction } from "@/lib/api";
import { displayPayeeName } from "@/lib/payee";
import { formatINR, cn } from "@/lib/utils";
import { suggestCategory, typeForCategory } from "@/lib/classify";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { statusLabel } from "@/components/classify-row";

export type ReviewGroup = {
  key: string;
  name: string;
  txns: Transaction[];
};

export function ReviewGroupCard({
  group,
  categories,
  onResolved,
}: {
  group: ReviewGroup;
  categories: string[];
  onResolved: (removedIds: string[], meta?: { applied_count: number; pattern?: string }) => void;
}) {
  const suggested = useMemo(() => suggestCategory(group.name), [group.name]);
  const [category, setCategory] = useState(suggested || "");
  const [saving, setSaving] = useState<"one" | "all" | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(group.txns.length <= 3);

  const totalPaise = group.txns.reduce((s, t) => s + t.amount_paise, 0);
  const n = group.txns.length;
  const primary = group.txns[0];

  async function save(mode: "one" | "all", txn: Transaction = primary) {
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
      if (mode === "all") {
        const pat = (res.pattern || group.name).toLowerCase();
        const removed = group.txns
          .filter(
            (t) =>
              t.id === txn.id ||
              t.original_description.toLowerCase().includes(pat) ||
              displayPayeeName(t.merchant, t.original_description).toLowerCase() ===
                group.key,
          )
          .map((t) => t.id);
        onResolved(removed.length ? removed : [txn.id], {
          applied_count: res.applied_count,
          pattern: res.pattern,
        });
      } else {
        onResolved([txn.id]);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(null);
    }
  }

  return (
    <div className="rounded-lg border border-amber-500/25 bg-card/40 p-4 sm:p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-base font-semibold tracking-tight text-foreground">
            {group.name}
          </h3>
          <p className="mt-1 text-sm text-muted-foreground">
            <span className="tabular-nums text-foreground/90">
              {n} txn{n === 1 ? "" : "s"}
            </span>
            <span className="mx-1.5">·</span>
            <span className="tabular-nums">{formatINR(totalPaise)}</span>
            {suggested && (
              <>
                <span className="mx-1.5">·</span>
                <span className="text-emerald-300/90">Suggested {suggested}</span>
              </>
            )}
          </p>
        </div>
        {n > 3 && (
          <button
            type="button"
            className="text-xs text-muted-foreground underline-offset-2 hover:underline"
            onClick={() => setExpanded((v) => !v)}
          >
            {expanded ? "Hide list" : "Show all"}
          </button>
        )}
      </div>

      <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="min-w-0 flex-1 space-y-1.5">
          <label className="text-xs text-muted-foreground">Category</label>
          <Select value={category || undefined} onValueChange={setCategory}>
            <SelectTrigger className="h-10 bg-background/50">
              <SelectValue placeholder="Choose category" />
            </SelectTrigger>
            <SelectContent>
              {categories.map((c) => (
                <SelectItem key={c} value={c}>
                  {c === "Other" ? "Uncategorized" : c}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={!category || saving !== null}
            onClick={() => save("one", primary)}
          >
            {saving === "one" ? "…" : n > 1 ? "This one only" : "Save"}
          </Button>
          <Button
            type="button"
            disabled={!category || saving !== null}
            onClick={() => save("all", primary)}
          >
            {saving === "all"
              ? "…"
              : n > 1
                ? `Apply to all ${n}`
                : `Apply to all “${group.name}”`}
          </Button>
        </div>
      </div>

      {suggested && category !== suggested && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          <button
            type="button"
            onClick={() => setCategory(suggested)}
            className={cn(
              "rounded-md border border-emerald-500/40 bg-emerald-500/10 px-2.5 py-1 text-xs text-emerald-200",
              "hover:bg-emerald-500/20",
            )}
          >
            Use {suggested}
          </button>
        </div>
      )}

      {err && <p className="mt-3 text-sm text-red-300">{err}</p>}

      {expanded && (
        <ul className="mt-4 space-y-2 border-t border-border/60 pt-3">
          {group.txns.map((t) => (
            <li
              key={t.id}
              className="flex flex-wrap items-center justify-between gap-2 rounded-md bg-background/30 px-3 py-2 text-sm"
            >
              <div className="min-w-0">
                <p className="text-xs text-muted-foreground">
                  {t.transaction_date.slice(0, 10)}
                  <span className="mx-1.5">·</span>
                  <Badge className="border-amber-500/30 text-[10px] text-amber-200/90">
                    {statusLabel(t)}
                  </Badge>
                </p>
                <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground" title={t.original_description}>
                  {t.original_description}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={cn(
                    "tabular-nums",
                    t.direction === "credit" ? "text-emerald-400" : "text-foreground",
                  )}
                >
                  {t.direction === "credit" ? "+" : "−"}
                  {formatINR(t.amount_paise)}
                </span>
                {n > 1 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2 text-xs"
                    disabled={!category || saving !== null}
                    onClick={() => save("one", t)}
                  >
                    This only
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
