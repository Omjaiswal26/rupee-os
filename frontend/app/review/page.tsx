"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { api, type Transaction } from "@/lib/api";
import { formatINR } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { CategoryCell, statusLabel } from "@/components/classify-row";

function ReviewInner() {
  const search = useSearchParams();
  const monthParam = search.get("month") || "";
  const [month, setMonth] = useState(monthParam);
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setMonth(monthParam);
  }, [monthParam]);

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

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Review</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Pending, needs review, Other, and uncategorized. Fix once — we save a rule.
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <label className="text-xs text-muted-foreground">Month</label>
        <input
          type="month"
          className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
          value={month}
          onChange={(e) => setMonth(e.target.value)}
        />
        {month && (
          <Link
            href={`/m/${month}`}
            className="text-xs text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
          >
            Open month
          </Link>
        )}
        {month && (
          <button
            type="button"
            className="text-xs text-muted-foreground underline-offset-2 hover:underline"
            onClick={() => setMonth("")}
          >
            All months
          </button>
        )}
      </div>

      {error && (
        <p className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : txns.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-sm text-muted-foreground">
            Nothing needs attention
            {month ? ` for ${month}` : ""}.
          </CardContent>
        </Card>
      ) : (
        <>
          <p className="text-xs text-muted-foreground">{txns.length} items</p>
          {txns.map((t) => (
            <Card key={t.id} className="border-amber-500/20">
              <CardHeader>
                <CardTitle className="text-base leading-snug text-foreground">
                  {t.original_description}
                </CardTitle>
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <span>{t.transaction_date.slice(0, 10)}</span>
                  <span>·</span>
                  <span className="tabular-nums">
                    {t.direction === "credit" ? "+" : "−"}
                    {formatINR(t.amount_paise)}
                  </span>
                  <Badge className="border-amber-500/40 text-amber-200">
                    {statusLabel(t)}
                  </Badge>
                  {t.category && t.category !== "Other" && (
                    <Badge>{t.category}</Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <CategoryCell
                  txn={t}
                  categories={categories}
                  onSaved={(updated) =>
                    setTxns((prev) => prev.filter((x) => x.id !== updated.id))
                  }
                />
              </CardContent>
            </Card>
          ))}
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
