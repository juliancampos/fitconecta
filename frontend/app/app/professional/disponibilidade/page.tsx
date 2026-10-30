import { Suspense } from "react";
import { connection } from "next/server";
import { AvailabilityView } from "@/components/professional/AvailabilityView";

async function Content() {
  await connection();
  return <AvailabilityView />;
}

export default function ProfessionalAvailabilityPage() {
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
