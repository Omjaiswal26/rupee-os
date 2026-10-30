"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { api, type Transaction } from "@/lib/api";
import {
  matchesReviewFilter,
  payeeGroupKey,
  type ReviewFilter,
} from "@/lib/classify";
import { displayPayeeName } from "@/lib/payee";
import { Card, CardContent } from "@/components/ui/card";
import { YearSelect } from "@/components/year-select";
import { ReviewGroupCard, type ReviewGroup } from "@/components/review-group-card";

function ReviewInner() {
  const search = useSearchParams();
  const monthParam = search.get("month") || "";
  const [month, setMonth] = useState(monthParam);
  const [year, setYear] = useState(() => {
    if (monthParam && /^\d{4}-\d{2}$/.test(monthParam)) {
      return Number(monthParam.slice(0, 4));
    }
    return new Date().getFullYear();
  });
  const [years, setYears] = useState<number[]>([year]);
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [filter, setFilter] = useState<ReviewFilter>("all");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setMonth(monthParam);
    if (monthParam && /^\d{4}-\d{2}$/.test(monthParam)) {
      setYear(Number(monthParam.slice(0, 4)));
    }
  }, [monthParam]);

  useEffect(() => {
    api
      .calendar(year)
      .then((r) => {
        if (r.years?.length) setYears(r.years);
      })
      .catch(() => {});
  }, [year]);

  const load = () => {
    setLoading(true);
    return Promise.all([api.review(month || undefined), api.categories()])
      .then(([r, c]) => {
        setTxns(r.transactions || []);
        setCategories(c.categories || []);
        setError(null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [month]);

  const filtered = useMemo(
    () => txns.filter((t) => matchesReviewFilter(t, filter)),
    [txns, filter],
  );

  const groups: ReviewGroup[] = useMemo(() => {
    const map = new Map<string, ReviewGroup>();
    for (const t of filtered) {
      const key = payeeGroupKey(t);
      const name = displayPayeeName(t.merchant, t.original_description);
      const g = map.get(key);
      if (g) g.txns.push(t);
      else map.set(key, { key, name, txns: [t] });
    }
    return [...map.values()].sort((a, b) => {
      if (b.txns.length !== a.txns.length) return b.txns.length - a.txns.length;
      const ba = b.txns.reduce((s, t) => s + t.amount_paise, 0);
      const aa = a.txns.reduce((s, t) => s + t.amount_paise, 0);
      return ba - aa;
    });
  }, [filtered]);

  const monthsInYear = useMemo(() => {
    return Array.from({ length: 12 }, (_, i) => {
      const m = String(i + 1).padStart(2, "0");
      return `${year}-${m}`;
    });
  }, [year]);

  function onResolved(removedIds: string[]) {
    const set = new Set(removedIds);
    setTxns((prev) => prev.filter((t) => !set.has(t.id)));
  }

  const filters: { key: ReviewFilter; label: string }[] = [
    { key: "all", label: "All" },
    { key: "uncategorized", label: "Uncategorized" },
    { key: "needs_review", label: "Needs review" },
    { key: "pending", label: "Pending" },
  ];

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Review</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Grouped by payee — pick a category, then save one or apply to all matches.
          </p>
        </div>
        <YearSelect
          year={year}
          years={years}
          onChange={(y) => {
            setYear(y);
            setMonth("");
          }}
          className="w-[7.5rem]"
          triggerClassName="bg-card/60"
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setMonth("")}
          className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
            !month
              ? "bg-primary text-primary-foreground"
              : "border border-border text-muted-foreground hover:bg-accent"
          }`}
        >
          All months
        </button>
        {monthsInYear.map((m) => {
          const label = new Date(`${m}-01`).toLocaleString("en-IN", { month: "short" });
          const active = month === m;
          return (
            <button
              key={m}
              type="button"
              onClick={() => setMonth(m)}
              className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
                active
                  ? "bg-primary text-primary-foreground"
                  : "border border-border text-muted-foreground hover:bg-accent"
              }`}
            >
              {label}
            </button>
          );
        })}
        {month && (
          <Link
            href={`/m/${month}`}
            className="text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
          >
            Open month
          </Link>
        )}
      </div>

      <div className="flex flex-wrap gap-1.5">
        {filters.map((f) => (
          <button
            key={f.key}
            type="button"
            onClick={() => setFilter(f.key)}
            className={`rounded-md px-2.5 py-1 text-xs transition-colors ${
              filter === f.key
                ? "bg-secondary text-foreground"
                : "border border-border text-muted-foreground hover:bg-accent"
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {error && (
        <p className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : filtered.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-sm text-muted-foreground">
            Nothing needs attention
            {month ? ` for ${month}` : ""}
            {filter !== "all" ? ` in “${filter}”` : ""}.
          </CardContent>
        </Card>
      ) : (
        <>
          <p className="text-xs text-muted-foreground">
            {filtered.length} transaction{filtered.length === 1 ? "" : "s"} · {groups.length}{" "}
            payee group{groups.length === 1 ? "" : "s"}
          </p>
          <div className="flex flex-col gap-4">
            {groups.map((g) => (
              <ReviewGroupCard
                key={g.key}
                group={g}
                categories={categories}
                onResolved={onResolved}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

export default function ReviewPage() {
  return (
    <Suspense fallback={<p className="text-sm text-muted-foreground">Loading…</p>}>
      <ReviewInner />
    </Suspense>
  );
}
