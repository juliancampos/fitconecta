import { Suspense } from "react";
import { connection } from "next/server";
import { DashboardView } from "@/components/professional/DashboardView";

async function Content() {
  await connection();
  return <DashboardView />;
}

export default function ProfessionalDashboardPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-6xl px-4 py-8">
          <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
        </div>
      }
    >
      <Content />
    </Suspense>
  );
}
