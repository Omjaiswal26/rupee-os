const RAIL =
  /^(WDL|TFR|DEP|UPI|IMPS|NEFT|RTGS|DR|CR|INFT|NWD|ATW|POS|PUR|GOL|PAY|PAYM|PAYMENT|SENT|NA|TO|FROM|BY)$/i;
const BANK =
  /^(HDFC|YESB|UTIB|ICIC|SBIN|AXIS|PAYTM|OKBIZ|P2M|P2A|AIRP|PUNB|KKBK|CNRB|IDFB|RATN|FDRL|MAHB|IOBA|BKID|BARB|UBIN|INDB)$/i;
const IFSC_LIKE = /^[A-Z]{4}0[A-Z0-9]{4,}$/i;
const OPAQUE_REF = /^[A-Z]{2,}\d{6,}$|^[A-Z0-9]{12,}$/i;
const DIGITS = /^\d{3,}$/;
const AT_BRANCH = /^AT\d*$/i;
const ADDRESS = /^(NEW|RAJINDER|NAGAR|DELHI|MUMBAI|BANGALORE|BENGALURU)$/i;

function splitNarration(s: string): string[] {
  return s.split(/[/\\|*\s]+/).map((p) => p.trim()).filter(Boolean);
}

function skipToken(up: string): boolean {
  return (
    RAIL.test(up) ||
    BANK.test(up) ||
    DIGITS.test(up) ||
    IFSC_LIKE.test(up) ||
    OPAQUE_REF.test(up) ||
    AT_BRANCH.test(up)
  );
}

function letterCount(s: string): number {
  return (s.match(/[A-Za-z]/g) || []).length;
}

function cleanToken(p: string): string {
  let t = p.replace(/^[,.;:]+|[,.;:]+$/g, "");
  if (t.includes("0") && letterCount(t) >= 3) {
    const repl = t.replace(/0/g, "O");
    if (/groww/i.test(repl)) return repl;
  }
  return t;
}

function fromParts(parts: string[], max = 3): string {
  const name: string[] = [];
  for (const p of parts) {
    if (!p) continue;
    const up = p.toUpperCase();
    if (skipToken(up) || ADDRESS.test(up)) {
      if (name.length > 0) break;
      continue;
    }
    if (letterCount(up) === 0) {
      if (name.length > 0) break;
      continue;
    }
    name.push(cleanToken(p));
    if (name.length >= max) break;
  }
  return name.join(" ");
}

/** Extract a short payee/merchant name from Indian bank narrations. */
export function extractPayeeName(description: string): string {
  if (!description) return "";

  // NEFT: counterparty usually after the last '*'.
  if (/NEFT/i.test(description) && description.includes("*")) {
    const segs = description.split("*");
    const last = (segs[segs.length - 1] || "").trim();
    const neft = fromParts(splitNarration(last), 4);
    if (neft) return neft;
  }

  return fromParts(splitNarration(description), 3);
}

const RAIL_MERCHANT = /^(DEP|WDL|TFR|UPI|NEFT|IMPS)$/i;

export function displayPayeeName(
  merchant: string | null | undefined,
  description: string,
): string {
  const m = (merchant || "").trim();
  // Ignore stale rail tokens saved as merchant (e.g. "DEP").
  if (m && !RAIL_MERCHANT.test(m)) return m;
  return extractPayeeName(description) || "—";
}
