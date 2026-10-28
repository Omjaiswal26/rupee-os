import Link from "next/link";

export default function HomePage() {
  return (
    <div className="mx-auto flex max-w-xl flex-col gap-6 pt-16">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">Rupee OS</h1>
        <p className="mt-3 leading-relaxed text-muted-foreground">
          Upload one calendar month of bank statement (XLSX or CSV). We classify
          locally — nothing leaves your machine.
        </p>
      </div>
      <div className="flex gap-3">
        <Link
          href="/upload"
          className="inline-flex h-9 items-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          Upload statement
        </Link>
        <Link
          href="/review"
          className="inline-flex h-9 items-center rounded-md border border-border px-4 text-sm text-muted-foreground hover:bg-accent"
        >
          Review queue
        </Link>
      </div>
    </div>
  );
}
