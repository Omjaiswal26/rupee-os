"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api, type HealthInsight, type MonthHealth } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

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

function valueTone(tone: string): string {
  switch (tone) {
    case "good":
      return "text-emerald-400";
    case "warn":
      return "text-amber-400";
    case "bad":
      return "text-red-400";
    default:
      return "text-foreground";
  }
}

function Metric({
  insight,
  onClick,
}: {
  insight: HealthInsight;
  onClick?: () => void;
}) {
  const interactive = Boolean(onClick);
  const Comp = interactive ? "button" : "div";
  return (
    <Comp
      type={interactive ? "button" : undefined}
      onClick={onClick}
      className={cn(
        "min-w-0 flex-1 rounded-md px-3 py-2.5 text-left",
        interactive && "transition-colors hover:bg-accent/50",
      )}
    >
      <p className="truncate text-[11px] text-muted-foreground">{insight.label}</p>
      <p className={cn("mt-1 text-lg font-semibold tabular-nums tracking-tight", valueTone(insight.tone))}>
        {insight.value}
      </p>
    </Comp>
  );
}

export function MonthHealthCard({
  month,
  classifying,
  enabled,
  onFocusCategory,
}: {
  month: string;
  classifying: boolean;
  enabled: boolean;
  onFocusCategory?: (category: string) => void;
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
            // not generated yet — need health model
          }
        }
        const ollama = await api.ollamaStatus().catch(() => null);
        if (ollama && !ollama.health_present) {
          setError(
            `Health model “${ollama.health_model || "?"}” not pulled. Run: ./scripts/pull-ollama-model.sh`,
          );
          setLoading(false);
          return;
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
        Month health appears when classification finishes.
      </div>
    );
  }

  if (loading && !report) {
    return (
      <div className="rounded-lg border border-border bg-card/40 px-5 py-6 text-sm text-muted-foreground">
        Generating…
      </div>
    );
  }

  if (error && !report) {
    const needsPull = /not pulled|health model|HEALTH_MODEL|pull-ollama/i.test(error);
    return (
      <div className="rounded-lg border border-border bg-card/40 px-5 py-5">
        <h2 className="text-sm font-medium text-foreground">Month health</h2>
        <p className="mt-2 text-sm text-muted-foreground">{error}</p>
        {needsPull ? (
          <p className="mt-3 text-xs text-muted-foreground">
            Health reports are fully model-based. Pull weights, then retry:
            <code className="mt-1.5 block rounded-md bg-muted/40 px-2 py-1.5 text-[11px] text-foreground">
              ./scripts/pull-ollama-model.sh
            </code>
          </p>
        ) : null}
        <button
          type="button"
          className="mt-4 text-xs font-medium text-foreground underline-offset-2 hover:underline"
          onClick={() => loadOrGenerate(true)}
        >
          Retry
        </button>
      </div>
    );
  }

  if (!report) return null;

  const tone = toneForVerdict(report.verdict);
  const s = report.signals;
  const reliable = s.reliable !== false && !s.unreliable_reason;
  const insights: HealthInsight[] = (s.insights?.length
    ? s.insights
    : [
        {
          id: "uncategorized",
          label: "Uncategorized",
          value: `${Math.round((s.uncategorized_ratio || 0) * 100)}%`,
          tone: (s.uncategorized_ratio || 0) >= 0.2 ? "bad" : "ok",
        },
        {
          id: "savings",
          label: "Savings",
          value:
            s.income_paise >= 10_000
              ? `${Math.round((s.savings_rate || 0) * 100)}%`
              : "—",
          tone: "ok",
        },
        {
          id: "invest",
          label: "Invested",
          value: `${Math.round((s.invest_share || 0) * 100)}%`,
          tone: (s.invest_share || 0) >= 0.15 ? "good" : "ok",
        },
      ]
  ).slice(0, 4);

  const drivers = s.drivers || [];
  const rawHeadline = (s.headline || report.tip || "").trim();
  // Reliable only: short headline. Unreliable: banner owns the message.
  const headline =
    reliable && rawHeadline && rawHeadline.split(/\s+/).length <= 14 ? rawHeadline : "";

  const barColor =
    tone === "good"
      ? "bg-emerald-400"
      : tone === "ok"
        ? "bg-sky-400"
        : tone === "warn"
          ? "bg-amber-400"
          : "bg-red-400";

  const verdictColor =
    tone === "good"
      ? "text-emerald-400"
      : tone === "ok"
        ? "text-sky-400"
        : tone === "warn"
          ? "text-amber-400"
          : "text-red-400";

  return (
    <section className="rounded-lg border border-border bg-card/40 px-5 py-5">
      {/* Header: title + score */}
      <div className="flex items-end justify-between gap-4">
        <div className="min-w-0">
          <h2 className="text-sm font-medium text-foreground">Month health</h2>
          <p className={cn("mt-0.5 text-xs", verdictColor)}>
            {report.verdict}
            {!reliable && (
              <span className="text-muted-foreground"> · provisional</span>
            )}
          </p>
        </div>
        <p
          className={cn(
            "text-4xl font-semibold tabular-nums leading-none tracking-tight text-foreground",
            !reliable && "text-foreground/50",
          )}
        >
          {report.score}
        </p>
      </div>

      <div className="mt-4 h-1 overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full transition-[width] duration-500", barColor)}
          style={{ width: `${Math.max(2, report.score)}%` }}
        />
      </div>

      {/* Reliability: one quiet line + two actions — no filled alert box */}
      {!reliable && (
        <div className="mt-5 flex flex-col gap-2 border-l-2 border-amber-500/70 pl-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
          <p className="text-sm text-foreground/90">
            Too much uncategorized — categorize, then regenerate.
          </p>
          <div className="flex shrink-0 items-center gap-3 text-xs">
            <button
              type="button"
              onClick={() => onFocusCategory?.("Other")}
              className="font-medium text-foreground underline-offset-2 hover:underline"
            >
              Show uncategorized
            </button>
            <Link
              href={`/review?month=${month}`}
              className="text-muted-foreground underline-offset-2 hover:text-foreground hover:underline"
            >
              Review
            </Link>
          </div>
        </div>
      )}

      {headline ? (
        <p className="mt-5 text-sm text-foreground/90">{headline}</p>
      ) : null}

      {/* Metrics: equal columns, color on value only */}
      <div className="mt-5 grid grid-cols-2 gap-px overflow-hidden rounded-md border border-border bg-border sm:grid-cols-4">
        {insights.map((ins) => (
          <div key={ins.id + ins.label} className="bg-card/80">
            <Metric
              insight={ins}
              onClick={
                onFocusCategory && (ins.id === "uncategorized" || ins.id === "top_category")
                  ? () =>
                      onFocusCategory(ins.id === "uncategorized" ? "Other" : ins.label)
                  : undefined
              }
            />
          </div>
        ))}
      </div>

      {drivers.length > 0 && (
        <p className="mt-3 text-[11px] tabular-nums text-muted-foreground">
          {drivers.map((d, i) => (
            <span key={d.label}>
              {i > 0 ? <span className="mx-1.5 text-border">·</span> : null}
              <span className={d.delta >= 0 ? "text-emerald-400/80" : "text-red-400/80"}>
                {d.delta >= 0 ? `+${d.delta}` : d.delta}
              </span>{" "}
              {d.label}
            </span>
          ))}
        </p>
      )}

      <div className="mt-5 flex items-center justify-between gap-3 border-t border-border/80 pt-3 text-[11px] text-muted-foreground">
        <span className="truncate">
          {report.model_name || "local"}
          {report.generated_at
            ? ` · ${new Date(report.generated_at).toLocaleString("en-IN", {
                day: "numeric",
                month: "short",
                hour: "2-digit",
                minute: "2-digit",
              })}`
            : ""}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 shrink-0 px-2 text-xs"
          disabled={loading}
          onClick={() => loadOrGenerate(true)}
        >
          {loading ? "…" : "Regenerate"}
        </Button>
      </div>
    </section>
  );
}
