import type { Transaction } from "@/lib/api";
import { displayPayeeName } from "@/lib/payee";

export function typeForCategory(
  category: string,
  direction: Transaction["direction"],
): string {
  if (category === "Transfers") return "transfer";
  if (category === "Investments") return "investment";
  if (category === "Fees & Charges") return "fee";
  if (category === "Income") return "income";
  return direction === "credit" ? "income" : "expense";
}

export function payeeGroupKey(t: Transaction): string {
  return displayPayeeName(t.merchant, t.original_description).toLowerCase();
}

/** Lightweight suggest from payee name (mirrors common merchant keywords). */
export function suggestCategory(payee: string): string | null {
  const p = payee.toLowerCase();
  const rules: [string, string][] = [
    ["groww", "Investments"],
    ["gr0ww", "Investments"],
    ["ppfas", "Investments"],
    ["zerodha", "Investments"],
    ["upstox", "Investments"],
    ["safe gold", "Investments"],
    ["safegold", "Investments"],
    ["swiggy", "Food & Dining"],
    ["zomato", "Food & Dining"],
    ["amazon", "Shopping"],
    ["flipkart", "Shopping"],
    ["uber", "Transport"],
    ["ola", "Transport"],
    ["irctc", "Transport"],
    ["netflix", "Entertainment"],
    ["spotify", "Entertainment"],
  ];
  for (const [needle, cat] of rules) {
    if (p.includes(needle)) return cat;
  }
  return null;
}

export type ReviewFilter = "all" | "uncategorized" | "needs_review" | "pending";

export function matchesReviewFilter(t: Transaction, f: ReviewFilter): boolean {
  if (f === "all") return true;
  if (f === "pending") return t.classification_status === "pending";
  if (f === "needs_review") return t.classification_status === "needs_review";
  if (f === "uncategorized") {
    return !t.category || t.category === "Other" || t.category === "";
  }
  return true;
}
