import type { CSSProperties } from "react";
import type { LucideIcon } from "lucide-react";
import {
  ArrowLeftRight,
  Banknote,
  Car,
  CircleHelp,
  Clapperboard,
  FileText,
  GraduationCap,
  HeartPulse,
  Home,
  LayoutGrid,
  Lightbulb,
  Plane,
  Receipt,
  Shield,
  ShoppingBag,
  ShoppingBasket,
  Sparkles,
  TrendingUp,
  UtensilsCrossed,
} from "lucide-react";
import { cn } from "@/lib/utils";

/** Stable palette (same as pie) keyed by category so chips & chart match. */
export const CATEGORY_PALETTE = [
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
] as const;

/** Order mirrors backend AllowedCategories for stable color assignment. */
const CATEGORY_ORDER = [
  "Income",
  "Food & Dining",
  "Groceries",
  "Transport",
  "Shopping",
  "Bills & Utilities",
  "Rent & Housing",
  "Healthcare",
  "Entertainment",
  "Travel",
  "Education",
  "Investments",
  "Transfers",
  "Fees & Charges",
  "Personal Care",
  "Insurance",
  "Taxes",
  "Other",
] as const;

const BY_CATEGORY: Record<string, LucideIcon> = {
  Income: Banknote,
  "Food & Dining": UtensilsCrossed,
  Groceries: ShoppingBasket,
  Transport: Car,
  Shopping: ShoppingBag,
  "Bills & Utilities": Lightbulb,
  "Rent & Housing": Home,
  Healthcare: HeartPulse,
  Entertainment: Clapperboard,
  Travel: Plane,
  Education: GraduationCap,
  Investments: TrendingUp,
  Transfers: ArrowLeftRight,
  "Fees & Charges": Receipt,
  "Personal Care": Sparkles,
  Insurance: Shield,
  Taxes: FileText,
  Other: CircleHelp,
  Uncategorized: CircleHelp,
};

export function categoryKey(category?: string | null): string {
  if (!category || category === "Other") return "Other";
  return category;
}

export function categoryColor(category?: string | null): string {
  const key = categoryKey(category);
  const idx = CATEGORY_ORDER.indexOf(key as (typeof CATEGORY_ORDER)[number]);
  if (idx >= 0) return CATEGORY_PALETTE[idx % CATEGORY_PALETTE.length];
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
  return CATEGORY_PALETTE[h % CATEGORY_PALETTE.length];
}

/** Soft chip tint: wash + border; pair with colored icon, muted label. */
export function categoryChipStyle(category?: string | null): {
  backgroundColor: string;
  borderColor: string;
} {
  const c = categoryColor(category);
  return {
    backgroundColor: `${c}14`,
    borderColor: `${c}55`,
  };
}

export function categoryAccent(category?: string | null): string {
  return categoryColor(category);
}

/** Amber attention tint (uncategorized / needs review). */
export const ATTENTION_CHIP_STYLE = {
  backgroundColor: "rgba(251, 191, 36, 0.10)",
  borderColor: "rgba(251, 191, 36, 0.40)",
} as const;

export const ATTENTION_ACCENT = "#fbbf24";

export function categoryIcon(category?: string | null): LucideIcon {
  const key = !category || category === "Other" ? "Uncategorized" : category;
  return BY_CATEGORY[key] || LayoutGrid;
}

export function CategoryIcon({
  category,
  className,
  strokeWidth = 2,
  style,
}: {
  category?: string | null;
  className?: string;
  strokeWidth?: number;
  style?: CSSProperties;
}) {
  const Icon = categoryIcon(category);
  return (
    <Icon
      className={cn("h-3.5 w-3.5 shrink-0", className)}
      strokeWidth={strokeWidth}
      style={style}
    />
  );
}
