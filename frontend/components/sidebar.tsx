"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { api, type MonthStatus, type MonthStatusInfo } from "@/lib/api";
import { cn } from "@/lib/utils";
import { YearSelect } from "@/components/year-select";

const statusDot: Record<MonthStatus, string> = {
  empty: "bg-zinc-600",
  pending: "bg-amber-400",
  needs_review: "bg-orange-400",
  ready: "bg-emerald-400",
};

const statusTitle: Record<MonthStatus, string> = {
  empty: "No statement",
  pending: "Uploaded — classifying",
  needs_review: "Needs review",
  ready: "Ready",
};

export function Sidebar() {
  const pathname = usePathname();
  const [year, setYear] = useState(new Date().getFullYear());
  const [years, setYears] = useState<number[]>([year]);
  const [months, setMonths] = useState<MonthStatusInfo[]>([]);

  useEffect(() => {
    api
      .calendar(year)
      .then((r) => {
        setMonths(r.months || []);
        if (r.years?.length) setYears(r.years);
      })
      .catch(() => setMonths([]));
  }, [year, pathname]);

  const overviewActive = pathname === "/";
  const reviewActive = pathname === "/review" || pathname.startsWith("/review/");

  return (
    <aside
      className={cn(
        "sticky top-0 z-20 flex h-svh shrink-0 flex-col self-start overflow-hidden",
        "border-r border-border bg-background/90 backdrop-blur-sm",
        "w-48 md:w-52 lg:w-56 xl:w-60 2xl:w-64",
      )}
    >
      <div className="shrink-0 border-b border-border px-3 py-3 lg:px-4 lg:py-4">
        <Link href="/" className="text-base font-semibold tracking-tight lg:text-lg">
          Rupee OS
        </Link>
        <p className="mt-0.5 text-xs text-muted-foreground">Financial health</p>
      </div>

      <nav className="flex shrink-0 flex-col gap-0.5 p-2 lg:p-3">
        <Link
          href="/"
          className={cn(
            "block rounded-md px-3 py-1.5 text-sm transition-colors",
            overviewActive
              ? "bg-secondary text-foreground"
              : "text-muted-foreground hover:bg-secondary/60 hover:text-foreground",
          )}
        >
          Overview
        </Link>
        <Link
          href="/review"
          className={cn(
            "block rounded-md px-3 py-1.5 text-sm transition-colors",
            reviewActive
              ? "bg-secondary text-foreground"
              : "text-muted-foreground hover:bg-secondary/60 hover:text-foreground",
          )}
        >
          Review
        </Link>
      </nav>

      <div className="mt-0.5 shrink-0 px-2 lg:px-3">
        <p className="mb-1 px-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Year
        </p>
        <YearSelect year={year} years={years} onChange={setYear} />
      </div>

      <div className="mt-2 shrink-0 px-3 text-xs font-medium uppercase tracking-wider text-muted-foreground lg:px-4">
        Months
      </div>
      <nav className="flex min-h-0 flex-1 flex-col overflow-hidden p-2 pt-1.5 lg:p-3 lg:pt-2">
        <ul className="flex flex-col gap-0.5">
          {months.map((m) => {
            const href = `/m/${m.month}`;
            const active = pathname === href;
            return (
              <li key={m.month}>
                <Link
                  href={href}
                  title={statusTitle[m.status]}
                  className={cn(
                    "flex items-center gap-2 rounded-md px-2.5 py-1 text-sm transition-colors lg:px-3 lg:py-1.5",
                    active
                      ? "bg-secondary text-foreground"
                      : "text-muted-foreground hover:bg-secondary/60 hover:text-foreground",
                    m.status === "empty" && !active && "opacity-50",
                  )}
                >
                  <span
                    className={cn("h-2 w-2 shrink-0 rounded-full", statusDot[m.status])}
                    aria-hidden
                  />
                  <span className="min-w-0 flex-1 truncate">{m.label}</span>
                  {m.transaction_count > 0 && (
                    <span className="shrink-0 text-[10px] tabular-nums text-muted-foreground">
                      {m.transaction_count}
                    </span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>

        <div className="mt-auto space-y-1 border-t border-border px-1 pt-2 text-[10px] text-muted-foreground">
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-zinc-600" /> No statement
          </div>
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-amber-400" /> Pending / classifying
          </div>
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-orange-400" /> Needs review
          </div>
          <div className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" /> Ready
          </div>
        </div>
      </nav>
    </aside>
  );
}
