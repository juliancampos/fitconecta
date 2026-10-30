"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { apiPost } from "@/lib/api";

const NAV = [
  { href: "/app/professional", label: "Painel" },
  { href: "/app/professional/disponibilidade", label: "Disponibilidade" },
  { href: "/app/professional/agendamentos", label: "Agendamentos" },
  { href: "/app/professional/financeiro", label: "Financeiro" },
  { href: "/app/professional/perfil", label: "Perfil" },
];

export function ProfessionalHeader() {
  const pathname = usePathname();
  const router = useRouter();

  async function handleLogout() {
    try {
      await apiPost("/api/v1/auth/logout", {});
    } catch {
      // Mesmo em caso de falha, removemos o acesso no cliente.
    }
    router.push("/auth/login");
    router.refresh();
  }

  return (
    <header className="sticky top-0 z-20 border-b border-slate-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-6 px-4">
        <Link
          href="/app/professional"
          className="flex items-center gap-2 text-lg font-extrabold text-ink"
        >
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand text-white">
            🦜
          </span>
          FitConecta<span className="-ml-2 text-brand">+</span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {NAV.map((item) => {
            const active =
              item.href === "/app/professional"
                ? pathname === item.href
                : pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`rounded-lg px-3 py-2 text-sm font-medium transition hover:bg-slate-100 ${
                  active ? "text-ink" : "text-muted"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="ml-auto flex items-center gap-3">
          <span className="rounded-full bg-brand-soft px-3 py-1 text-xs font-semibold text-brand">
            Profissional
          </span>
          <button
            type="button"
            onClick={handleLogout}
            className="rounded-lg px-3 py-2 text-sm font-medium text-muted transition hover:bg-slate-100"
          >
            Sair
          </button>
        </div>
      </div>

      <nav className="flex items-center gap-1 overflow-x-auto border-t border-slate-100 px-4 py-2 md:hidden">
        {NAV.map((item) => {
          const active =
            item.href === "/app/professional"
              ? pathname === item.href
              : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`whitespace-nowrap rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                active ? "bg-slate-100 text-ink" : "text-muted"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
    </header>
  );
}

export default ProfessionalHeader;
