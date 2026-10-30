"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiPost } from "@/lib/api";

export function Header() {
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
          href="/app/client"
          className="flex items-center gap-2 text-lg font-extrabold text-ink"
        >
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand text-white">
            🦜
          </span>
          FitConecta<span className="-ml-2 text-brand">+</span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          <Link
            href="/app/client"
            className="rounded-lg px-3 py-2 text-sm font-medium text-ink transition hover:bg-slate-100"
          >
            Buscar profissionais
          </Link>
          <Link
            href="/app/client/agendamentos"
            className="rounded-lg px-3 py-2 text-sm font-medium text-muted transition hover:bg-slate-100"
          >
            Meus agendamentos
          </Link>
        </nav>

        <div className="ml-auto flex items-center gap-3">
          <button
            type="button"
            onClick={handleLogout}
            className="rounded-lg px-3 py-2 text-sm font-medium text-muted transition hover:bg-slate-100"
          >
            Sair
          </button>
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-soft text-sm font-bold text-brand">
              JS
            </div>
            <div className="hidden leading-tight sm:block">
              <p className="text-sm font-semibold text-ink">João Silva</p>
              <p className="text-xs text-muted">Cliente</p>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}

export default Header;
