import { Suspense } from "react";
import { connection } from "next/server";
import { AppointmentsView } from "@/components/professional/AppointmentsView";

async function Content() {
  await connection();
  return <AppointmentsView />;
}

export default function ProfessionalAppointmentsPage() {
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
