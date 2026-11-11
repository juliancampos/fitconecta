import { AdminNav } from "@/components/admin/AdminNav";

export default function PainelLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[240px_1fr]">
      <aside className="border-b border-slate-200 bg-surface lg:border-b-0 lg:border-r">
        <div className="lg:sticky lg:top-0 lg:h-screen">
          <AdminNav />
        </div>
      </aside>
      <main className="min-w-0">
        <div className="mx-auto max-w-[1400px] p-4 sm:p-6 lg:p-8">{children}</div>
      </main>
    </div>
  );
}
