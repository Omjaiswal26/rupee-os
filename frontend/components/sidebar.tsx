"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { api, type MonthStatus, type MonthStatusInfo } from "@/lib/api";
import { cn } from "@/lib/utils";

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

  const navLink = (href: string, label: string) => {
    const active = pathname === href || pathname.startsWith(href + "/");
    return (
      <Link
        href={href}
        className={cn(
          "block rounded-md px-3 py-2 text-sm transition-colors",
          active
            ? "bg-secondary text-foreground"
            : "text-muted-foreground hover:bg-secondary/60 hover:text-foreground",
        )}
      >
        {label}
      </Link>
    );
  };

  return (
    <aside className="flex h-full w-56 shrink-0 flex-col border-r border-border bg-background/80">
      <div className="border-b border-border px-4 py-5">
        <Link href="/" className="text-lg font-semibold tracking-tight">
          Rupee OS
        </Link>
        <p className="mt-1 text-xs text-muted-foreground">Financial health</p>
      </div>

      <nav className="flex flex-col gap-1 p-3">
        {navLink("/upload", "Upload")}
        {navLink("/review", "Review")}
      </nav>

      <div className="mt-1 px-3">
        <label className="mb-1.5 block px-1 text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Year
        </label>
        <select
          className="h-9 w-full rounded-md border border-input bg-transparent px-2 text-sm"
          value={year}
          onChange={(e) => setYear(Number(e.target.value))}
        >
          {years.map((y) => (
            <option key={y} value={y} className="bg-card">
              {y}
            </option>
          ))}
        </select>
      </div>

      <div className="mt-3 px-4 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        Months
      </div>
      <nav className="flex-1 overflow-y-auto p-3 pt-2">
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
                    "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
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
                  <span className="flex-1">{m.label}</span>
                  {m.transaction_count > 0 && (
                    <span className="text-[10px] tabular-nums text-muted-foreground">
                      {m.transaction_count}
                    </span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>

        <div className="mt-4 space-y-1.5 border-t border-border px-1 pt-3 text-[10px] text-muted-foreground">
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
