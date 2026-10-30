import { Suspense } from "react";
import { connection } from "next/server";
import { ProfileView } from "@/components/professional/ProfileView";

async function Content() {
  await connection();
  return <ProfileView />;
}

export default function ProfessionalProfilePage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-3xl px-4 py-8">
          <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
        </div>
      }
    >
      <Content />
    </Suspense>
  );
}
