import { redirect } from "next/navigation";

/** Standalone upload removed — upload only from a month page. */
export default async function UploadRedirectPage({
  searchParams,
}: {
  searchParams: Promise<{ month?: string }>;
}) {
  const { month } = await searchParams;
  if (month && /^\d{4}-\d{2}$/.test(month)) {
    redirect(`/m/${month}`);
  }
  redirect("/");
}
