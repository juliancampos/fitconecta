import { Suspense } from "react";
import { connection } from "next/server";
import { FinanceView } from "@/components/professional/FinanceView";

async function Content() {
  await connection();
  return <FinanceView />;
}

export default function ProfessionalFinancePage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-5xl px-4 py-8">
          <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
        </div>
      }
    >
      <Content />
    </Suspense>
  );
}
