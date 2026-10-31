"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api, type YearDashboard, type YearHealth } from "@/lib/api";
import { cn, formatINR, formatPercent, formatSavingsRate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { YearSelect } from "@/components/year-select";

function scoreColor(score: number | null | undefined) {
  if (score == null) return "bg-zinc-700";
  if (score >= 80) return "bg-emerald-400";
  if (score >= 60) return "bg-sky-400";
  if (score >= 40) return "bg-amber-400";
  return "bg-red-400";
}

function verdictTone(verdict: string) {
  switch (verdict) {
    case "Strong":
      return "text-emerald-300";
    case "Stable":
      return "text-sky-300";
    case "Needs attention":
      return "text-amber-200";
    default:
      return "text-red-300";
  }
}

export default function HomePage() {
  const [year, setYear] = useState(new Date().getFullYear());
  const [dash, setDash] = useState<YearDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [healthBusy, setHealthBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api
      .dashboard(year)
      .then((d) => {
        setDash(d);
        setError(null);
        if (d.years?.length && !d.years.includes(year)) {
          // keep selected year even if empty
        }
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load"))
      .finally(() => setLoading(false));
  }, [year]);

  useEffect(() => {
    load();
  }, [load]);

  async function ensureHealth(force: boolean) {
    if (!dash || dash.months_with_data === 0) return;
    if (!force && dash.health) return;
    setHealthBusy(true);
    try {
      const h = await api.generateYearHealth(year);
      setDash((prev) => (prev ? { ...prev, health: h } : prev));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Health generation failed");
    } finally {
      setHealthBusy(false);
    }
  }

  useEffect(() => {
    if (!dash || dash.health || dash.months_with_data === 0 || healthBusy) return;
    ensureHealth(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dash?.year, dash?.health, dash?.months_with_data]);

  const years = useMemo(() => {
    const list = dash?.years?.length ? [...dash.years] : [year];
    if (!list.includes(year)) list.unshift(year);
    return list.sort((a, b) => b - a);
  }, [dash, year]);

  const maxScore = 100;
  const health: YearHealth | null = dash?.health ?? null;

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">Overview</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Year health and monthly scores — local only.
          </p>
        </div>
        <YearSelect
          year={year}
          years={years}
          onChange={setYear}
          className="w-[7.5rem]"
          triggerClassName="bg-card/60"
        />
      </div>

      {error && (
        <p className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-red-300">
          {error}
        </p>
      )}

      {loading && !dash ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : !dash || dash.months_with_data === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-start gap-3 p-8">
            <p className="text-sm text-muted-foreground">
              No statements for {year} yet. Open a month in the sidebar to upload.
            </p>
            <Link
              href={`/m/${year}-01`}
              className="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground"
            >
              Open January {year}
            </Link>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <Stat label="Income" value={formatINR(dash.income_paise)} />
            <Stat label="Expenses" value={formatINR(dash.expense_paise)} />
            <Stat
              label="Uncategorized"
              value={formatINR(dash.uncategorized_paise)}
              tone={dash.uncategorized_paise > 0 ? "bad" : undefined}
            />
          </div>

          <YearHealthPanel
            year={year}
            health={health}
            busy={healthBusy}
            onRegenerate={() => ensureHealth(true)}
          />

          {(dash.needs_review_months > 0 ||
            (dash.needs_review_count || 0) > 0 ||
            (dash.pending_count || 0) > 0) && (
            <p className="rounded-md border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100/90">
              {dash.needs_review_months} month(s) still have items to review
              {(dash.needs_review_count || 0) > 0 && ` · ${dash.needs_review_count} flagged`}
              {(dash.pending_count || 0) > 0 && ` · ${dash.pending_count} pending`}
              {" · "}
              <Link href="/review" className="underline underline-offset-2">
                Open review queue
              </Link>
            </p>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Score by month · {year}</CardTitle>
              <p className="text-xs text-muted-foreground">
                Click a month to open it. Empty months have no statement.
              </p>
            </CardHeader>
            <CardContent>
              <div className="flex items-end gap-2 sm:gap-3">
                {dash.months.map((m) => {
                  const score = m.score ?? null;
                  const h = score != null ? Math.max(8, (score / maxScore) * 120) : 8;
                  return (
                    <Link
                      key={m.month}
                      href={`/m/${m.month}`}
                      className="group flex min-w-0 flex-1 flex-col items-center gap-2"
                      title={
                        m.has_statement
                          ? `${m.label}: score ${score ?? "—"}`
                          : `${m.label}: no statement`
                      }
                    >
                      <span className="text-[10px] tabular-nums text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 sm:opacity-100">
                        {score ?? "—"}
                      </span>
                      <div className="flex h-[120px] w-full items-end justify-center">
                        <div
                          className={cn(
                            "w-full max-w-[28px] rounded-t-md transition-opacity group-hover:opacity-90",
                            m.has_statement ? scoreColor(score) : "bg-zinc-800",
                            !m.has_statement && "opacity-40",
                          )}
                          style={{ height: m.has_statement ? h : 8 }}
                        />
                      </div>
                      <span className="text-[10px] text-muted-foreground">{m.label}</span>
                    </Link>
                  );
                })}
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function YearHealthPanel({
  year,
  health,
  busy,
  onRegenerate,
}: {
  year: number;
  health: YearHealth | null;
  busy: boolean;
  onRegenerate: () => void;
}) {
  if (busy && !health) {
    return (
      <div className="rounded-lg border border-border bg-card/40 px-5 py-6 text-sm text-muted-foreground">
        Generating {year} health…
      </div>
    );
  }
  if (!health) {
    return (
      <div className="rounded-lg border border-border bg-card/40 px-5 py-4 text-sm text-muted-foreground">
        No year health yet.{" "}
        <button type="button" className="underline" onClick={onRegenerate}>
          Generate
        </button>
      </div>
    );
  }

  const s = health.signals;
  return (
    <div className="rounded-lg border border-border bg-card/40 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-sm font-medium text-foreground">{year} health</h2>
          <p className={cn("mt-1 text-xs font-medium", verdictTone(health.verdict))}>
            {health.verdict}
          </p>
        </div>
        <div className="text-right">
          <p className="text-3xl font-semibold tabular-nums tracking-tight">{health.score}</p>
          <p className="text-[10px] text-muted-foreground">Score</p>
        </div>
      </div>
      <div className="mt-3 h-2 overflow-hidden rounded-full bg-secondary">
        <div
          className={cn("h-full rounded-full", scoreColor(health.score))}
          style={{ width: `${health.score}%` }}
        />
      </div>
      <p className="mt-4 text-sm leading-relaxed text-foreground/90">{health.tip}</p>
      <div className="mt-4 grid grid-cols-3 gap-2">
        <Mini
          label="Savings"
          value={formatSavingsRate(s.savings_rate, s.income_paise)}
          hint={s.income_paise < 10_000 ? "low income" : undefined}
        />
        <Mini label="Invested" value={formatPercent(s.invest_share)} />
        <Mini label="Uncategorized" value={formatPercent(s.uncategorized_ratio)} />
      </div>
      <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span>
          Stored · {health.model_name} ·{" "}
          {health.generated_at ? new Date(health.generated_at).toLocaleString() : "—"}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-xs"
          disabled={busy}
          onClick={onRegenerate}
        >
          {busy ? "…" : "Regenerate"}
        </Button>
      </div>
    </div>
  );
}

function Mini({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-md border border-border/70 bg-background/40 px-3 py-2">
      <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-sm font-medium tabular-nums">{value}</p>
      {hint ? <p className="text-[10px] text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "bad";
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p
          className={cn(
            "text-2xl font-semibold tracking-tight tabular-nums",
            tone === "bad" && "text-red-300",
          )}
        >
          {value}
        </p>
      </CardContent>
    </Card>
  );
}
