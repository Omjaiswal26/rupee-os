"use client";

import { useEffect, useState } from "react";
import { api, type OllamaStatus } from "@/lib/api";

export function OllamaStatusBanner() {
  const [st, setSt] = useState<OllamaStatus | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .ollamaStatus()
      .then((r) => {
        if (!cancelled) setSt(r);
      })
      .catch(() => {
        if (!cancelled) {
          setSt({
            ok: false,
            reachable: false,
            classify_model: "",
            classify_present: false,
            health_model: "",
            health_present: false,
            hint: "API unreachable — is docker compose up?",
            error: "fetch failed",
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (!st) return null;
  if (st.ok && st.classify_present) return null;

  const msg = !st.reachable
    ? st.hint || "Ollama is not reachable from the API."
    : st.hint || `Model ${st.classify_model || "?"} is not pulled yet.`;

  return (
    <div className="border-b border-amber-500/30 bg-amber-500/[0.07] px-5 py-2.5 text-sm text-amber-100/95 md:px-6 lg:px-8 xl:px-10">
      <p className="font-medium text-amber-50">Classifier not ready</p>
      <p className="mt-0.5 text-xs text-amber-100/80">
        {msg}{" "}
        <code className="rounded bg-background/40 px-1.5 py-0.5 text-[11px] text-foreground/90">
          ./scripts/pull-ollama-model.sh
        </code>
      </p>
    </div>
  );
}
