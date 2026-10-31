"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, type ImportRecord } from "@/lib/api";
import { formatMonthLabel } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileDropzone } from "@/components/file-dropzone";

/** Upload / replace statement for a specific calendar month only. */
export function MonthUpload({
  month,
  replacing = false,
  onDone,
}: {
  month: string;
  replacing?: boolean;
  onDone?: () => void;
}) {
  const router = useRouter();
  const [file, setFile] = useState<File | null>(null);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [current, setCurrent] = useState<ImportRecord | null>(null);

  useEffect(() => {
    if (!current || current.status === "completed" || current.status === "failed") return;
    const t = setInterval(() => {
      api
        .importOne(current.id)
        .then((r) => {
          setCurrent(r.import);
          if (r.import.status === "completed") {
            onDone?.();
            const m = r.import.validated_month?.slice(0, 7);
            if (m) router.push(`/m/${m}`);
          }
        })
        .catch(() => {});
    }, 1500);
    return () => clearInterval(t);
  }, [current, onDone, router]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file || !month) return;
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await api.upload(file, password, month);
      setSuccess(res.message);
      setCurrent(res.import);
      setFile(null);
      setPassword("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {replacing ? "Replace statement" : "Upload statement"} · {formatMonthLabel(month)}
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          One calendar month of bank XLSX/CSV. Password-protected XLSX supported — password is
          never stored.
          {replacing ? " Existing transactions for this month will be removed first." : ""}
        </p>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="flex flex-col gap-4">
          <FileDropzone file={file} onFile={setFile} disabled={busy} />

          <div className="space-y-1.5">
            <label className="text-xs text-muted-foreground" htmlFor={`pw-${month}`}>
              XLSX password <span className="text-muted-foreground/70">(only if protected)</span>
            </label>
            <Input
              id={`pw-${month}`}
              type="password"
              autoComplete="off"
              placeholder="Leave blank for CSV or unlocked XLSX"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={busy}
            />
          </div>

          <Button type="submit" disabled={!file || busy} className="w-full sm:w-auto">
            {busy ? "Uploading…" : replacing ? "Replace & process" : "Upload & process"}
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
            {current.error_message && <p className="text-red-300">{current.error_message}</p>}
            {current.status === "completed" && current.validated_month && (
              <Link
                href={`/m/${current.validated_month.slice(0, 7)}`}
                className="text-primary hover:underline"
              >
                Open month →
              </Link>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
