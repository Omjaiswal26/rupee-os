"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api, type MonthSummary, type Transaction } from "@/lib/api";
import { cn, formatINR, formatMonthLabel } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CategoryPie } from "@/components/category-pie";
import { CategoryCell } from "@/components/classify-row";

type Chip = { key: string; label: string; kind: "all" | "ambiguous" | "category" };
type SortKey = "date" | "amount" | "description" | "category";
type SortOrder = "asc" | "desc";

const PAGE_SIZE = 50;

export default function MonthPage() {
  const params = useParams<{ month: string }>();
  const month = params.month;
  const [summary, setSummary] = useState<MonthSummary | null>(null);
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [total, setTotal] = useState(0);
  const [categories, setCategories] = useState<string[]>([]);
  const [q, setQ] = useState("");
  const [qDebounced, setQDebounced] = useState("");
  /** "" = all, "__ambiguous__" = attention queue, else category name */
  const [chip, setChip] = useState("");
  const [sort, setSort] = useState<SortKey>("date");
  const [order, setOrder] = useState<SortOrder>("desc");
  const [page, setPage] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionMsg, setActionMsg] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setQDebounced(q), 250);
    return () => clearTimeout(t);
  }, [q]);

  // Reset to first page when filters/search change.
  useEffect(() => {
    setPage(0);
  }, [qDebounced, chip]);

  const statusFilter = chip === "__ambiguous__" ? "attention" : undefined;
  const categoryFilter =
    chip && chip !== "__ambiguous__" ? chip : undefined;

  const load = useCallback(() => {
    setLoading(true);
    return Promise.all([
      api.summary(month),
      api.transactions(month, {
        q: qDebounced,
        status: statusFilter,
        category: categoryFilter,
        sort,
        order,
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      }),
      api.categories(),
    ])
      .then(([s, t, c]) => {
        setSummary(s);
        setTxns(t.transactions || []);
        setTotal(t.total || 0);
        setCategories(c.categories || []);
        setError(null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, [month, qDebounced, statusFilter, categoryFilter, sort, order, page]);

  useEffect(() => {
    load();
  }, [load]);

  const chips: Chip[] = useMemo(() => {
    const present = (summary?.category_breakdown || [])
      .filter((c) => c.amount_paise > 0)
      .map((c) => c.category);
    const out: Chip[] = [
      { key: "", label: "All", kind: "all" },
      { key: "__ambiguous__", label: "Ambiguous", kind: "ambiguous" },
    ];
    for (const cat of present) {
      out.push({
        key: cat,
        label: cat === "Other" ? "Uncategorized" : cat,
        kind: "category",
      });
    }
    return out;
  }, [summary]);

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const from = total === 0 ? 0 : page * PAGE_SIZE + 1;
  const to = Math.min(total, (page + 1) * PAGE_SIZE);

  function toggleSort(key: SortKey) {
    if (sort === key) {
      setOrder((o) => (o === "asc" ? "desc" : "asc"));
    } else {
      setSort(key);
      setOrder(key === "description" || key === "category" ? "asc" : "desc");
    }
    setPage(0);
  }

  async function onReclassify() {
    setBusy(true);
    setActionMsg(null);
    try {
      const res = await api.reclassify(month);
      setActionMsg(res.message + ` (${res.reset_count} reset)`);
      setPage(0);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Reclassify failed");
    } finally {
      setBusy(false);
    }
  }

  if (loading && !summary) {
    return <p className="text-sm text-muted-foreground">Loading…</p>;
  }
  if (error && !summary) {
    return <p className="text-sm text-red-300">{error}</p>;
  }

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{formatMonthLabel(month)}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {summary?.transaction_count || 0} transactions
            {(summary?.needs_review_count || 0) > 0 && (
              <span> · {summary?.needs_review_count} need review</span>
            )}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link
            href={`/review?month=${month}`}
            className="inline-flex h-9 items-center rounded-md border border-border px-4 text-sm hover:bg-accent"
          >
            Review queue
          </Link>
          <Link
            href={`/upload?month=${month}`}
            className="inline-flex h-9 items-center rounded-md border border-border px-4 text-sm hover:bg-accent"
          >
            Re-upload
          </Link>
          <Button
            variant="secondary"
            disabled={busy || (summary?.transaction_count || 0) === 0}
            onClick={onReclassify}
          >
            {busy ? "Working…" : "Reclassify"}
          </Button>
        </div>
      </div>

      {actionMsg && (
        <p className="rounded-md border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-emerald-300">
          {actionMsg}
        </p>
      )}
      {error && (
        <p className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {(summary?.transaction_count || 0) === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-start gap-3 p-8">
            <p className="text-sm text-muted-foreground">No statement for this month yet.</p>
            <Link
              href={`/upload?month=${month}`}
              className="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground"
            >
              Upload statement
            </Link>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <Stat label="Income" value={formatINR(summary?.income_paise || 0)} />
            <Stat label="Expenses" value={formatINR(summary?.expense_paise || 0)} />
            <Stat
              label="Uncategorized"
              value={formatINR(summary?.uncategorized_paise || 0)}
              tone={(summary?.uncategorized_paise || 0) > 0 ? "bad" : undefined}
            />
          </div>

          {summary && (
            <CategoryPie data={summary.category_breakdown || []} title="By category" />
          )}

          <Card>
            <CardHeader className="space-y-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <CardTitle>Transactions</CardTitle>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {total === 0 ? "No matches" : `${from}–${to} of ${total}`}
                  </p>
                </div>
                <Input
                  className="max-w-xs"
                  placeholder="Search description…"
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                />
              </div>

              <div className="flex flex-wrap gap-1.5">
                {chips.map((f) => {
                  const active = chip === f.key;
                  const ambiguous = f.kind === "ambiguous";
                  return (
                    <button
                      key={f.key || "all"}
                      type="button"
                      onClick={() => setChip(f.key)}
                      className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                        active
                          ? ambiguous
                            ? "bg-amber-500/90 text-amber-950"
                            : "bg-primary text-primary-foreground"
                          : ambiguous
                            ? "border border-amber-500/40 text-amber-200 hover:bg-amber-500/10"
                            : "border border-border text-muted-foreground hover:bg-accent hover:text-foreground"
                      }`}
                    >
                      {f.label}
                    </button>
                  );
                })}
              </div>
            </CardHeader>
            <CardContent className="overflow-x-auto p-0">
              {txns.length === 0 ? (
                <p className="p-5 text-sm text-muted-foreground">No transactions match filters</p>
              ) : (
                <table className="w-full text-left text-sm">
                  <thead className="border-b border-border text-xs text-muted-foreground">
                    <tr>
                      <SortTh
                        label="Date"
                        active={sort === "date"}
                        order={order}
                        onClick={() => toggleSort("date")}
                      />
                      <SortTh
                        label="Description"
                        active={sort === "description"}
                        order={order}
                        onClick={() => toggleSort("description")}
                      />
                      <SortTh
                        label="Category"
                        active={sort === "category"}
                        order={order}
                        onClick={() => toggleSort("category")}
                      />
                      <SortTh
                        label="Amount"
                        active={sort === "amount"}
                        order={order}
                        align="right"
                        onClick={() => toggleSort("amount")}
                      />
                    </tr>
                  </thead>
                  <tbody>
                    {txns.map((t) => (
                      <tr key={t.id} className="border-b border-border/50 last:border-0">
                        <td className="whitespace-nowrap px-5 py-3 text-muted-foreground">
                          {t.transaction_date.slice(0, 10)}
                        </td>
                        <td className="max-w-md truncate px-5 py-3" title={t.original_description}>
                          {t.original_description}
                        </td>
                        <td className="px-5 py-3">
                          <CategoryCell
                            txn={t}
                            categories={categories}
                            onSaved={(updated, meta) => {
                              if (meta && meta.applied_count > 1) {
                                load();
                                return;
                              }
                              setTxns((prev) => {
                                if (chip === "__ambiguous__") {
                                  return prev.filter((x) => x.id !== updated.id);
                                }
                                return prev.map((x) =>
                                  x.id === updated.id ? updated : x,
                                );
                              });
                              api.summary(month).then(setSummary).catch(() => {});
                            }}
                          />
                        </td>
                        <td
                          className={`whitespace-nowrap px-5 py-3 text-right tabular-nums ${
                            t.direction === "credit" ? "text-emerald-400" : ""
                          }`}
                        >
                          {t.direction === "credit" ? "+" : "−"}
                          {formatINR(t.amount_paise)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}

              {total > PAGE_SIZE && (
                <div className="flex items-center justify-between gap-3 border-t border-border px-5 py-3">
                  <p className="text-xs text-muted-foreground">
                    Page {page + 1} of {pageCount}
                  </p>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={page === 0 || loading}
                      onClick={() => setPage((p) => Math.max(0, p - 1))}
                    >
                      Previous
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={page + 1 >= pageCount || loading}
                      onClick={() => setPage((p) => p + 1)}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function SortTh({
  label,
  active,
  order,
  onClick,
  align = "left",
}: {
  label: string;
  active: boolean;
  order: SortOrder;
  onClick: () => void;
  align?: "left" | "right";
}) {
  return (
    <th className={cn("px-5 py-3 font-medium", align === "right" && "text-right")}>
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "inline-flex items-center gap-1 transition-colors hover:text-foreground",
          align === "right" && "flex-row-reverse",
          active ? "text-foreground" : "text-muted-foreground",
        )}
      >
        {label}
        <span className="w-3 text-[10px] tabular-nums opacity-80">
          {active ? (order === "asc" ? "↑" : "↓") : ""}
        </span>
      </button>
    </th>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "good" | "bad";
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p
          className={`text-2xl font-semibold tracking-tight tabular-nums ${
            tone === "good" ? "text-emerald-400" : tone === "bad" ? "text-red-300" : ""
          }`}
        >
          {value}
        </p>
      </CardContent>
    </Card>
  );
}
