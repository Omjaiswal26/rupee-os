# Test / sample statements

Synthetic India-style narrations for local demos and parser tests.

Payees and amounts are deliberately fake (`DEMOEATS`, `FAKECAB`, cartoonish ₹ values). **No real accounts or people.**

| File | Purpose |
|------|---------|
| `sample_single_month.csv` | Happy path — one calendar month (Oct 2026). Upload this in the UI. |
| `sample_multi_month.csv` | Negative test — spans two months; import should be rejected. |

Upload via UI (month page) or:

```bash
curl -F file=@backend/testdata/sample_single_month.csv http://localhost:8080/imports
```
