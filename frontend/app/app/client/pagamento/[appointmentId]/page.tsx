import { Suspense } from "react";
import { connection } from "next/server";
import { PaymentView } from "@/components/client/PaymentView";

async function Content() {
  await connection();
  return <PaymentView />;
}

export default function PagamentoPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-2xl px-4 py-8">
          <div className="h-64 animate-pulse rounded-2xl bg-slate-200" />
        </div>
      }
    >
      <Content />
    </Suspense>
  );
}
