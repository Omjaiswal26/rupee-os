"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { api, type ImportRecord } from "@/lib/api";
import { formatMonthLabel } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function UploadPage() {
  return (
    <Suspense fallback={<p className="text-sm text-muted-foreground">Loading…</p>}>
      <UploadForm />
    </Suspense>
  );
}

function UploadForm() {
  const search = useSearchParams();
  const replaceMonth = search.get("month") || "";

  const [file, setFile] = useState<File | null>(null);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [current, setCurrent] = useState<ImportRecord | null>(null);
  const [imports, setImports] = useState<ImportRecord[]>([]);

  const refresh = () =>
    api.imports().then((r) => setImports(r.imports || [])).catch(() => {});

  useEffect(() => {
    refresh();
  }, []);

  useEffect(() => {
    if (!current || current.status === "completed" || current.status === "failed") return;
    const t = setInterval(() => {
      api
        .importOne(current.id)
        .then((r) => {
          setCurrent(r.import);
          if (r.import.status === "completed" || r.import.status === "failed") {
            refresh();
          }
        })
        .catch(() => {});
    }, 1500);
    return () => clearInterval(t);
  }, [current]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) return;
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.upload(file, password, replaceMonth);
      setSuccess(res.message);
      setCurrent(res.import);
      setFile(null);
      setPassword("");
      refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          {replaceMonth ? "Re-upload" : "Upload"}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          One calendar month only. Bank XLSX/CSV with Date, Narration, and Debit/Credit.
          Password-protected XLSX is supported — password is never stored. Same file can be
          re-uploaded.
        </p>
      </div>

      {replaceMonth && (
        <p className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-200">
          Replacing data for <strong>{formatMonthLabel(replaceMonth)}</strong>. Existing
          transactions for that month will be removed first.
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Statement file</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <Input
              type="file"
              accept=".csv,.xlsx,.xls"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
            <div className="space-y-1.5">
              <label className="text-xs text-muted-foreground" htmlFor="stmt-password">
                XLSX password (if protected)
              </label>
              <Input
                id="stmt-password"
                type="password"
                autoComplete="off"
                placeholder="Statement password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <Button type="submit" disabled={!file || busy}>
              {busy ? "Uploading…" : replaceMonth ? "Replace & process" : "Upload & process"}
            </Button>
          </form>
          {error && (
            <p className="mt-4 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-red-300">
              {error}
            </p>
          )}
          {success && (
            <p className="mt-4 rounded-md border border-primary/30 bg-primary/10 px-3 py-2 text-sm text-emerald-300">
              {success}
            </p>
          )}
          {current && (
            <div className="mt-4 space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">Status</span>
                <Badge>{current.status}</Badge>
              </div>
              {current.error_message && (
                <p className="text-red-300">{current.error_message}</p>
              )}
              {current.status === "completed" && current.validated_month && (
                <Link
                  href={`/m/${current.validated_month.slice(0, 7)}`}
                  className="text-primary hover:underline"
                >
                  Open {current.validated_month.slice(0, 7)} dashboard →
                </Link>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Recent imports</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {imports.length === 0 ? (
            <p className="text-sm text-muted-foreground">None yet</p>
          ) : (
            imports.slice(0, 8).map((imp) => (
              <div
                key={imp.id}
                className="flex items-center justify-between gap-3 rounded-md border border-border/60 px-3 py-2 text-sm"
              >
                <div className="min-w-0">
                  <p className="truncate">{imp.filename}</p>
                  <p className="text-xs text-muted-foreground">
                    {imp.validated_month?.slice(0, 7) || "—"} · {imp.transaction_count} txns
                  </p>
                </div>
                <Badge>{imp.status}</Badge>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
