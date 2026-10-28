"use client";

import { formatINR } from "@/lib/utils";

const COLORS = [
  "#34d399",
  "#60a5fa",
  "#fbbf24",
  "#f472b6",
  "#a78bfa",
  "#2dd4bf",
  "#fb923c",
  "#94a3b8",
  "#e879f9",
  "#4ade80",
];

type Slice = { category: string; amount_paise: number };

export function CategoryPie({
  data,
  title = "By category",
}: {
  data: Slice[];
  title?: string;
}) {
  const slices = (data || []).filter((d) => d.amount_paise > 0);
  const total = slices.reduce((s, d) => s + d.amount_paise, 0);

  if (total <= 0) {
    return (
      <div className="rounded-lg border border-border bg-card/40 p-6 text-sm text-muted-foreground">
        No categories yet.
      </div>
    );
  }

  const r = 42;
  const c = 2 * Math.PI * r;
  let offset = 0;

  const arcs = slices.map((d, i) => {
    const frac = d.amount_paise / total;
    const len = frac * c;
    const item = {
      ...d,
      color: COLORS[i % COLORS.length],
      dash: `${len} ${c - len}`,
      offset,
      pct: Math.round(frac * 1000) / 10,
    };
    offset -= len;
    return item;
  });

  return (
    <div className="rounded-lg border border-border bg-card/40 p-5">
      <h2 className="text-sm font-medium text-foreground">{title}</h2>
      <div className="mt-4 flex flex-col items-center gap-6 sm:flex-row sm:items-start">
        <div className="relative shrink-0">
          <svg width="168" height="168" viewBox="0 0 100 100" aria-hidden>
            <circle
              cx="50"
              cy="50"
              r={r}
              fill="none"
              stroke="hsl(var(--border))"
              strokeWidth="12"
            />
            {arcs.map((a) => (
              <circle
                key={a.category}
                cx="50"
                cy="50"
                r={r}
                fill="none"
                stroke={a.color}
                strokeWidth="12"
                strokeDasharray={a.dash}
                strokeDashoffset={a.offset}
                strokeLinecap="butt"
                transform="rotate(-90 50 50)"
              />
            ))}
            <text
              x="50"
              y="48"
              textAnchor="middle"
              className="fill-muted-foreground"
              style={{ fontSize: "6px" }}
            >
              Total
            </text>
            <text
              x="50"
              y="58"
              textAnchor="middle"
              className="fill-foreground"
              style={{ fontSize: "7px", fontWeight: 600 }}
            >
              {formatINR(total).replace("₹", "₹")}
            </text>
          </svg>
        </div>
        <ul className="w-full min-w-0 flex-1 space-y-2">
          {arcs.slice(0, 10).map((a) => (
            <li key={a.category} className="flex items-center gap-2 text-sm">
              <span
                className="h-2.5 w-2.5 shrink-0 rounded-full"
                style={{ background: a.color }}
              />
              <span className="min-w-0 flex-1 truncate text-muted-foreground">
                {a.category}
              </span>
              <span className="shrink-0 tabular-nums text-foreground/90">
                {a.pct}%
              </span>
              <span className="w-24 shrink-0 text-right tabular-nums">
                {formatINR(a.amount_paise)}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
