const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type ImportRecord = {
  id: string;
  filename: string;
  file_type: string;
  validated_month: string | null;
  status: string;
  transaction_count: number;
  error_code?: string | null;
  error_message?: string | null;
  created_at: string;
};

export type Transaction = {
  id: string;
  import_id: string;
  transaction_date: string;
  original_description: string;
  amount_paise: number;
  currency: string;
  direction: "debit" | "credit";
  transaction_type?: string | null;
  category?: string | null;
  subcategory?: string | null;
  classification_status: string;
};

export type MonthSummary = {
  month: string;
  income_paise: number;
  expense_paise: number;
  net_paise: number;
  transaction_count: number;
  needs_review_count: number;
  category_breakdown: { category: string; amount_paise: number }[];
};

export type MonthStatus = "empty" | "pending" | "needs_review" | "ready";

export type MonthStatusInfo = {
  month: string;
  label: string;
  status: MonthStatus;
  transaction_count: number;
  pending_count: number;
  needs_review_count: number;
  has_statement: boolean;
};

export type YearCalendar = {
  year: number;
  years: number[];
  months: MonthStatusInfo[];
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    cache: "no-store",
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || data.message || `Request failed (${res.status})`);
  }
  return data as T;
}

export const api = {
  months: () => request<{ months: string[] }>("/months"),
  calendar: (year: number) =>
    request<YearCalendar>(`/months/calendar?year=${year}`),
  summary: (month: string) => request<MonthSummary>(`/months/${month}/summary`),
  transactions: (
    month: string,
    opts: {
      q?: string;
      category?: string;
      status?: string;
      sort?: "date" | "amount" | "description" | "category";
      order?: "asc" | "desc";
      limit?: number;
      offset?: number;
    } = {},
  ) => {
    const params = new URLSearchParams();
    params.set("limit", String(opts.limit ?? 50));
    params.set("offset", String(opts.offset ?? 0));
    params.set("sort", opts.sort ?? "date");
    params.set("order", opts.order ?? "desc");
    if (opts.q) params.set("q", opts.q);
    if (opts.category) params.set("category", opts.category);
    if (opts.status) params.set("status", opts.status);
    return request<{
      transactions: Transaction[];
      total: number;
      limit: number;
      offset: number;
      sort: string;
      order: string;
    }>(`/months/${month}/transactions?${params.toString()}`);
  },
  reclassify: (month: string) =>
    request<{ reset_count: number; message: string }>(`/months/${month}/reclassify`, {
      method: "POST",
    }),
  clearMonth: (month: string) =>
    request<{ cleared: boolean }>(`/months/${month}`, { method: "DELETE" }),
  review: (month = "") => {
    const params = new URLSearchParams();
    params.set("limit", "200");
    if (month) params.set("month", month);
    return request<{ transactions: Transaction[]; total: number }>(
      `/transactions/review?${params.toString()}`,
    );
  },
  imports: () => request<{ imports: ImportRecord[] }>("/imports"),
  importOne: (id: string) => request<{ import: ImportRecord }>(`/imports/${id}`),
  categories: () =>
    request<{ categories: string[]; types: string[] }>("/meta/categories"),
  upload: async (file: File, password = "", replaceMonth = "") => {
    const body = new FormData();
    body.append("file", file);
    body.append("password", password);
    if (replaceMonth) body.append("replace_month", replaceMonth);
    return request<{ import: ImportRecord; message: string }>("/imports", {
      method: "POST",
      body,
    });
  },
  correct: (
    id: string,
    payload: {
      category: string;
      subcategory?: string;
      type?: string;
      note?: string;
      create_rule?: boolean;
    },
  ) =>
    request<{ transaction: Transaction }>(`/transactions/${id}/classification`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),
};
