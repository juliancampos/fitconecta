import { Suspense } from "react";
import { connection } from "next/server";
import { HomeView } from "@/components/HomeView";

async function HomeContent() {
  // A disponibilidade é relativa à data atual do usuário.
  await connection();
  return <HomeView />;
}

function HomeFallback() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="h-8 w-64 animate-pulse rounded-lg bg-slate-200" />
      <div className="mt-6 h-28 animate-pulse rounded-2xl bg-slate-200" />
      <div className="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div
            key={i}
            className="h-56 animate-pulse rounded-2xl bg-slate-200"
          />
        ))}
      </div>
    </div>
  );
}

export default function HomePage() {
  return (
    <Suspense fallback={<HomeFallback />}>
      <HomeContent />
    </Suspense>
  );
}
