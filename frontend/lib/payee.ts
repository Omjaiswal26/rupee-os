/** Extract a short payee/merchant name from Indian UPI/IMPS bank narrations. */
export function extractPayeeName(description: string): string {
  if (!description) return "";
  const rail =
    /^(WDL|TFR|UPI|IMPS|NEFT|RTGS|DR|CR|INFT|NWD|ATW|POS|PUR|GOL|PAY|PAYM|PAYMENT)$/i;
  const bank =
    /^(HDFC|YESB|UTIB|ICIC|SBIN|AXIS|PAYTM|OKBIZ|P2M|P2A|AIRP)$/i;
  const parts = description.split(/[/\\|/\s]+/).map((p) => p.trim()).filter(Boolean);
  const name: string[] = [];
  for (const p of parts) {
    if (p.length < 1) continue;
    if (rail.test(p) || bank.test(p) || /^\d{3,}$/.test(p)) {
      if (name.length > 0) break;
      continue;
    }
    if (/^AT\d/i.test(p) || (/^AT/i.test(p) && p.length <= 6)) continue;
    if (!/[A-Za-z]/.test(p)) {
      if (name.length > 0) break;
      continue;
    }
    name.push(p);
    if (name.length >= 3) break;
  }
  return name.join(" ");
}

export function displayPayeeName(
  merchant: string | null | undefined,
  description: string,
): string {
  const m = (merchant || "").trim();
  if (m) return m;
  return extractPayeeName(description) || "—";
}
