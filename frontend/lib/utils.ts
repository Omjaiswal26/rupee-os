import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatINR(paise: number): string {
  const rupees = paise / 100;
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2,
  }).format(rupees);
}

export function formatMonthLabel(yyyyMm: string): string {
  const [y, m] = yyyyMm.split("-").map(Number);
  return new Date(y, m - 1, 1).toLocaleString("en-IN", {
    month: "long",
    year: "numeric",
  });
}

/** Ratio 0–1 → percent label; clamps wild values. */
export function formatPercent(ratio: number, digits = 0): string {
  if (!Number.isFinite(ratio)) return "—";
  const pct = ratio * 100;
  if (Math.abs(pct) > 999) return pct < 0 ? "< −999%" : "> 999%";
  return `${pct.toFixed(digits)}%`;
}

/**
 * Savings rate is (income − expense) / income.
 * Meaningless when recognized income is ~0 (shows absurd negatives).
 */
export function formatSavingsRate(rate: number, incomePaise = 0): string {
  if (!Number.isFinite(rate)) return "—";
  if (incomePaise > 0 && incomePaise < 10_000) return "—"; // < ₹100 income
  if (incomePaise <= 0 && rate <= 0) return "—";
  return formatPercent(rate);
}
