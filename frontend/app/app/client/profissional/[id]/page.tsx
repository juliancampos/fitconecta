import { Suspense } from "react";
import { connection } from "next/server";
import { ProfessionalDetailView } from "@/components/ProfessionalDetailView";

async function ProfessionalContent() {
  await connection();
  return <ProfessionalDetailView />;
}

export default function ProfessionalPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-4xl px-4 py-8">
          <div className="h-80 animate-pulse rounded-2xl bg-slate-200" />
        </div>
      }
    >
      <ProfessionalContent />
    </Suspense>
  );
}
