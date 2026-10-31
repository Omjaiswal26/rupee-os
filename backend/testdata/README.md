# Test / sample statements

Synthetic India-style narrations for local demos and parser tests. **No real accounts or people.**

| File | Purpose |
|------|---------|
| `sample_single_month.csv` | Happy path — one calendar month (Oct 2026). Upload this in the UI. |
| `sample_multi_month.csv` | Negative test — spans two months; import should be rejected. |
| `bank_stmt.xlsx` | Placeholder path for Excel fixtures (add a real synthetic XLSX in tests if needed). |

Upload via UI (month page) or:

```bash
curl -F file=@backend/testdata/sample_single_month.csv http://localhost:8080/imports
```
