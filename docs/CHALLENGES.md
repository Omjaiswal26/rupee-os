# Challenges building a local AI bank classifier (Rupee OS)

Notes from building an India-first personal finance tracker that classifies bank statement rows with **Ollama only** (no cloud LLM). Useful raw material for a future LinkedIn / blog post.

---

## 1. Password-protected Excel statements

**Symptom:** Banks export “XLSX” that isn’t a normal zip-based workbook — OLE encryption. Upload failed even with the password typed in the UI.

**Cause:** Password wasn’t always reaching the unlock path; excelize needs `Decrypt` before parse; Docker images needed rebuild so the API/worker actually had the new code.

**Lesson:** Indian bank exports are messier than CSV demos. Treat password + format as a first-class product path, not an edge case.

---

## 2. Header / column detection across banks

**Symptom:** “Header row not found” on real statements.

**Cause:** Narration / debit / credit / balance column names vary wildly (`Details`, `Particulars`, `Withdrawal Amt.`, etc.).

**Lesson:** Alias tables + sample-row diagnostics beat assuming one bank’s template.

---

## 3. Model not pulled → silent “pending forever”

**Symptom:** Hundreds of `classify failed` with `model '…' not found` (404). UI stayed yellow / pending. No `failed` status.

**Cause:** Ollama container was healthy; the **weights** weren’t pulled into the volume yet. On error we intentionally left rows as `pending` for retry — but nothing auto-retried after pull.

**Lesson:** Preflight “is model present?” before enqueue. Distinguish `pending` vs `classification_failed`. Document one-shot `pull` in onboarding.

---

## 4. Wrong model for the job: Qwen3.5 4B (thinking / multimodal)

**Symptom:** First classify run after pull: CPU ~15 tok/s, `n_gen` climbing past 1600 tokens, then:

`context deadline exceeded (Client.Timeout exceeded while awaiting headers)`

Almost zero successful classifications. UI: 205 txns, ~200 pending, everything “Uncategorized”.

**Cause:** `qwen3.5:4b` is oriented toward reasoning / vision. With only `format: "json"` and no `num_predict`, it dumped long chain-of-thought instead of a tiny JSON object. HTTP client timeout was 120s; one txn never finished.

**Solution we shipped:**
- Switch default to **`qwen2.5:3b`** (smaller, instruction-following, no thinking tax).
- Cap generation: `num_predict: 128`, `temperature: 0`.
- Pass a **JSON Schema** (enums for category/type) as Ollama `format`.
- Shorten the prompt; log each classify result.

**Lesson for LinkedIn:** “Local AI” isn’t pick-a-model-and-go. Matching model *behavior* to the task (short structured label vs chat/reasoning) matters more than parameter count. A 3B non-thinking model beat a 4B thinking model on bank labels + latency.

---

## 5. Everything classified as “Transfers” (accuracy)

**Symptom (real Oct 2026 statement):**

| Narration cue | Expected | Got |
|---------------|----------|-----|
| `TOY CORNER` | Shopping / Entertainment | Transfers |
| `MOM HAND` (food stall) | Food & Dining | Transfers |
| `SAFE GOLD` | Investments | Transfers |

Most UPI debits land on **Transfers** with high confidence and reasoning like “UPI transaction for transferring money…”.

**Why the model does this (not “model is dumb” alone):**

1. **Bank text is payment-rail noise, not merchant English.**  
   Strings look like:  
   `WDL TFR UPI/DR/…/TOY CORNER/HDFC/vyapa r.173/UP … NEW RAJINDER NAGAR`  
   Dominant tokens: `WDL`, `TFR`, `UPI`, `DR`, bank codes, account/branch junk. The merchant name is a short, often truncated, substring (`MOM HAND`, `SAFE GOL`). A small LLM latches onto **UPI = transfer**.

2. **`Transfers` is a valid category and a valid `type`.**  
   Taxonomy has both category `Transfers` and type `transfer`. Without hard rules (“UPI debit to a merchant ≠ peer transfer”), the model’s safest prior for any `UPI/DR` is transfer.

3. **Prompt doesn’t teach Indian merchant heuristics.**  
   Current prompt only lists enums + “JSON only”. It never says:  
   - Prefer merchant/purpose over rail (`UPI`/`IMPS`/`NEFT`).  
   - `*GOLD*`, digi-gold apps → Investments.  
   - Food names / Swiggy / Zomato → Food.  
   - Toy / retail → Shopping.  
   - True Transfers = self accounts, family names, “To self”, credit↔debit mirrors.

4. **`temperature: 0` + weak signal → sticky wrong answer.**  
   Once the model decides “UPI ⇒ Transfers”, greedy decoding repeats that pattern for similar rows (almost every debit on an Indian statement is UPI).

5. **No merchant dictionary / rules layer yet.**  
   Product design intended user rules + corrections; without seeded rules for Swiggy / Groww / SafeGold / etc., every row is a cold LLM call on noisy text.

6. **Type vs category conflation.**  
   Logs showed Food & Dining with `type: transfer` (Swiggy). Schema allows that combo. Humans care about spend category; the model optimizes for “how money moved.”

**Not fixing yet — candidate solutions for later:**
- Prompt: “Ignore WDL/TFR/UPI/bank codes; classify by merchant token.”
- Few-shot Indian UPI examples (toy shop, food stall, digi-gold, rent).
- Soft rules / keyword map before LLM (SafeGold → Investments, SWIGGY → Food).
- Disallow category=Transfers unless description looks like P2P/self (no merchant brand).
- Human correction → auto rule (already partly in product).

---

## 6. Why classification still feels slow

Even after the model swap (~2s per txn on CPU):

- **One HTTP call per transaction** (no batching). 205 txns ≈ several minutes wall clock.
- **CPU-only Ollama** in Docker on Apple Silicon (no GPU passthrough in typical Compose setup).
- **Cold start / model load** on first request after idle.
- **Sequential worker loop** — no parallel classify with concurrency limits.
- Earlier (Qwen3.5): timeouts made it *infinitely* slow; now it’s “slow but finishing.”

**Future levers:** batch N narrations per prompt, rule short-circuit, keep model warm, optional GPU, parallel workers with rate limit.

---

## 7. Product / UX traps we hit

- Sidebar “200 pending” vs 205 real pending — UI counted only the loaded page of transactions.
- Re-upload / replace month / reclassify needed because failed classifies stayed pending.
- Income/expense totals can look fine while categories are all wrong (rail vs purpose).

---

## Narrative arc (for a post)

1. Built local-first finance OS so bank data never leaves the machine.  
2. Real Indian XLSX + passwords broke the happy path.  
3. Pulled a “modern” 4B model — it thought forever and timed out.  
4. Switched to a smaller instruction model + JSON schema + token caps — pipeline ran.  
5. Then the harder truth: **UPI narrations are hostile to naive LLMs**; they look like transfers. Accuracy needs India-specific prompts, rules, and human-in-the-loop — not a bigger model.  
6. Local AI is a product constraint (privacy, cost) and an engineering constraint (latency, model choice, eval on real statements).

---

## Changelog of relevant fixes (so far)

| When | Change |
|------|--------|
| Early | XLSX password decrypt, broader column aliases |
| Mid | Month calendar statuses, re-upload, reclassify |
| Late | Replace `qwen3.5:4b` → `qwen2.5:3b`; `num_predict`; JSON schema; result logs |
| Prompt v2 | Hard rules + few-shots; type↔category normalize (Transfers only with type transfer) |
| Open | Keyword seed rules; remaining Transfer over-prediction; speed via batching |

---

*Private build notes — sanitize any personal merchant/amount detail before posting publicly.*
