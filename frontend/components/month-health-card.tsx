"use client";

import { useCallback, useEffect, useState } from "react";
import { api, type MonthHealth } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn, formatPercent, formatSavingsRate } from "@/lib/utils";

function toneForVerdict(verdict: string): "good" | "ok" | "warn" | "bad" {
  switch (verdict) {
    case "Strong":
      return "good";
    case "Stable":
      return "ok";
    case "Needs attention":
      return "warn";
    default:
      return "bad";
  }
}

function Signal({
  label,
  value,
  hint,
}: {
  label: string;
  value: string;
  hint: string;
}) {
  return (
    <div className="rounded-md border border-border/70 bg-background/40 px-3 py-2">
      <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-sm font-medium tabular-nums text-foreground">{value}</p>
      <p className="text-[10px] text-muted-foreground">{hint}</p>
    </div>
  );
}

export function MonthHealthCard({
  month,
  classifying,
  enabled,
}: {
  month: string;
  classifying: boolean;
  enabled: boolean;
}) {
  const [report, setReport] = useState<MonthHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadOrGenerate = useCallback(
    async (force: boolean) => {
      if (!enabled || classifying) return;
      setLoading(true);
      setError(null);
      try {
        if (!force) {
          try {
            const existing = await api.health(month);
            setReport(existing);
            setLoading(false);
            return;
          } catch {
            // not generated yet
          }
        }
        const generated = await api.generateHealth(month);
        setReport(generated);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load health report");
      } finally {
        setLoading(false);
      }
    },
    [month, enabled, classifying],
  );

  useEffect(() => {
    setReport(null);
    if (enabled && !classifying) {
      loadOrGenerate(false);
    }
  }, [month, enabled, classifying, loadOrGenerate]);

  if (!enabled) return null;

  if (classifying) {
    return (
      <div className="rounded-lg border border-border bg-card/40 px-5 py-4 text-sm text-muted-foreground">
        Month health will appear when classification finishes.
      </div>
    );
  }

  if (loading && !report) {
    return (
      <div className="rounded-lg border border-border bg-card/40 px-5 py-6 text-sm text-muted-foreground">
        Generating month health…
      </div>
    );
  }

  if (error && !report) {
    return (
      <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-5 py-4 text-sm">
        <p className="text-red-300">{error}</p>
        <Button
          type="button"
          variant="secondary"
          size="sm"
          className="mt-3"
          onClick={() => loadOrGenerate(true)}
        >
          Retry
        </Button>
      </div>
    );
  }

  if (!report) return null;

  const tone = toneForVerdict(report.verdict);
  const s = report.signals;
  const barColor =
    tone === "good"
      ? "bg-emerald-400"
      : tone === "ok"
        ? "bg-sky-400"
        : tone === "warn"
          ? "bg-amber-400"
          : "bg-red-400";

  return (
    <div className="rounded-lg border border-border bg-card/40 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-medium text-foreground">Month health</h2>
          <p
            className={cn(
              "mt-1 text-xs font-medium",
              tone === "good" && "text-emerald-300",
              tone === "ok" && "text-sky-300",
              tone === "warn" && "text-amber-200",
              tone === "bad" && "text-red-300",
            )}
          >
            {report.verdict}
          </p>
        </div>
        <div className="text-right">
          <p className="text-3xl font-semibold tabular-nums tracking-tight">{report.score}</p>
          <p className="text-[10px] text-muted-foreground">Score</p>
        </div>
      </div>

      <div className="mt-3 h-2 overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full transition-[width] duration-500", barColor)}
          style={{ width: `${report.score}%` }}
        />
      </div>

      <p className="mt-4 text-sm leading-relaxed text-foreground/90">{report.tip}</p>

      <div className="mt-4 grid grid-cols-3 gap-2">
        <Signal
          label="Savings"
          value={formatSavingsRate(s.savings_rate, s.income_paise)}
          hint={
            s.income_paise < 10_000
              ? "low income"
              : s.savings_rate >= 0.1
                ? "ok"
                : "watch"
          }
        />
        <Signal
          label="Invested"
          value={formatPercent(s.invest_share)}
          hint={s.invest_share >= 0.15 ? "strong" : "low"}
        />
        <Signal
          label="Uncategorized"
          value={formatPercent(s.uncategorized_ratio)}
          hint={s.uncategorized_ratio >= 0.15 ? "watch" : "ok"}
        />
      </div>
      {(s.needs_review_count || 0) > 0 && (
        <p className="mt-3 text-xs text-amber-200/80">
          {s.needs_review_count} need review
        </p>
      )}

      <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span>
          Stored · {report.model_name || "local"} ·{" "}
          {report.generated_at ? new Date(report.generated_at).toLocaleString() : "—"}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          disabled={loading}
          onClick={() => loadOrGenerate(true)}
        >
          {loading ? "…" : "Regenerate"}
        </Button>
      </div>
    </div>
  );
}
