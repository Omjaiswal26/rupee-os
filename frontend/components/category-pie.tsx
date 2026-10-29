"use client";

import { formatINR, cn } from "@/lib/utils";

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

function polar(cx: number, cy: number, r: number, angleDeg: number) {
  const a = ((angleDeg - 90) * Math.PI) / 180;
  return { x: cx + r * Math.cos(a), y: cy + r * Math.sin(a) };
}

/** Donut wedge path from startAngle→endAngle (degrees, clockwise from top). */
function donutSlice(
  cx: number,
  cy: number,
  rInner: number,
  rOuter: number,
  start: number,
  end: number,
) {
  const large = end - start > 180 ? 1 : 0;
  const o0 = polar(cx, cy, rOuter, start);
  const o1 = polar(cx, cy, rOuter, end);
  const i0 = polar(cx, cy, rInner, end);
  const i1 = polar(cx, cy, rInner, start);
  return [
    `M ${o0.x} ${o0.y}`,
    `A ${rOuter} ${rOuter} 0 ${large} 1 ${o1.x} ${o1.y}`,
    `L ${i0.x} ${i0.y}`,
    `A ${rInner} ${rInner} 0 ${large} 0 ${i1.x} ${i1.y}`,
    "Z",
  ].join(" ");
}

export function CategoryPie({
  data,
  title = "By category",
  activeCategory = "",
  onSelectCategory,
}: {
  data: Slice[];
  title?: string;
  activeCategory?: string;
  onSelectCategory?: (category: string) => void;
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

  let angle = 0;
  const arcs = slices.map((d, i) => {
    const frac = d.amount_paise / total;
    // Leave a hairline gap except for full circle single slice.
    const sweep = Math.min(359.999, frac * 360);
    const start = angle;
    const end = angle + sweep;
    angle = end;
    return {
      ...d,
      color: COLORS[i % COLORS.length],
      start,
      end,
      pct: Math.round(frac * 1000) / 10,
    };
  });

  const cx = 50;
  const cy = 50;
  const rOuter = 48;
  const rInner = 32;

  return (
    <div className="rounded-lg border border-border bg-card/40 p-5">
      <h2 className="text-sm font-medium text-foreground">{title}</h2>
      <div className="mt-4 flex flex-col items-center gap-6 sm:flex-row sm:items-start">
        <div className="relative shrink-0">
          <svg width="168" height="168" viewBox="0 0 100 100" className="overflow-visible">
            {arcs.map((a) => {
              const active = activeCategory === a.category;
              const muted = Boolean(activeCategory) && !active;
              const path =
                a.end - a.start >= 359.9
                  ? // full ring: two semicircles
                    [
                      `M ${cx} ${cy - rOuter}`,
                      `A ${rOuter} ${rOuter} 0 1 1 ${cx} ${cy + rOuter}`,
                      `A ${rOuter} ${rOuter} 0 1 1 ${cx} ${cy - rOuter}`,
                      `M ${cx} ${cy - rInner}`,
                      `A ${rInner} ${rInner} 0 1 0 ${cx} ${cy + rInner}`,
                      `A ${rInner} ${rInner} 0 1 0 ${cx} ${cy - rInner}`,
                      "Z",
                    ].join(" ")
                  : donutSlice(cx, cy, rInner, rOuter, a.start, a.end);
              return (
                <path
                  key={a.category}
                  d={path}
                  fill={a.color}
                  fillRule="evenodd"
                  opacity={muted ? 0.28 : active ? 1 : 0.92}
                  stroke={active ? "hsl(var(--foreground))" : "transparent"}
                  strokeWidth={active ? 1.2 : 0}
                  className={cn(
                    "outline-none focus:outline-none",
                    onSelectCategory && "cursor-pointer transition-opacity hover:opacity-100",
                    active && "drop-shadow-sm",
                  )}
                  onClick={() => onSelectCategory?.(a.category)}
                >
                  <title>{`${a.category}: ${a.pct}%`}</title>
                </path>
              );
            })}
            <text
              x="50"
              y="48"
              textAnchor="middle"
              className="fill-muted-foreground pointer-events-none"
              style={{ fontSize: "6px" }}
            >
              Total
            </text>
            <text
              x="50"
              y="58"
              textAnchor="middle"
              className="fill-foreground pointer-events-none"
              style={{ fontSize: "7px", fontWeight: 600 }}
            >
              {formatINR(total)}
            </text>
          </svg>
        </div>
        <ul className="w-full min-w-0 flex-1 space-y-1">
          {arcs.slice(0, 10).map((a) => {
            const active = activeCategory === a.category;
            const muted = Boolean(activeCategory) && !active;
            return (
              <li key={a.category}>
                <button
                  type="button"
                  onClick={() => onSelectCategory?.(a.category)}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-1.5 py-1.5 text-left text-sm transition-colors",
                    onSelectCategory && "hover:bg-accent/60",
                    active && "bg-secondary",
                    muted && "opacity-50",
                  )}
                >
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
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    </div>
  );
}
