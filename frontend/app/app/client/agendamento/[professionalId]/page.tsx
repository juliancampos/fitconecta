import { Suspense } from "react";
import { connection } from "next/server";
import { AgendamentoView } from "@/components/client/AgendamentoView";

async function Content() {
  await connection();
  return <AgendamentoView />;
}

export default function AgendamentoPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-2xl px-4 py-8">
          <div className="h-72 animate-pulse rounded-2xl bg-slate-200" />
        </div>
      }
    >
      <Content />
    </Suspense>
  );
}
