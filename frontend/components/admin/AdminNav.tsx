"use client";

import Link from "next/link";
import Image from "next/image";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";
import { apiPost } from "@/lib/admin/api";

const links = [
  { href: "/admin/agendamentos", label: "Agendamentos" },
  { href: "/admin/pagamentos", label: "Pagamentos" },
  { href: "/admin/profissionais", label: "Profissionais" },
  { href: "/admin/clientes", label: "Clientes" },
];

export function AdminNav() {
  const pathname = usePathname();
  const router = useRouter();
  const [leaving, setLeaving] = useState(false);

  async function logout() {
    setLeaving(true);
    try {
      await apiPost("/api/v1/admin/auth/logout");
    } catch {
      // Mesmo com falha de rede, o cookie expira no navegador ao navegar.
    } finally {
      router.replace("/admin/login");
      router.refresh();
    }
  }

  return (
    <div className="flex h-full flex-col justify-between gap-6 p-4 lg:p-6">
      <div>
        <Link href="/admin/agendamentos" className="flex items-center gap-2">
          <Image src="/images/logo-fitconecta.png" alt="" width={36} height={36} className="h-9 w-9 rounded-xl" />
          <span className="text-lg font-extrabold text-ink">
            FitConecta<span className="-ml-1 text-brand">+</span>
          </span>
        </Link>
        <p className="mt-1 text-xs font-semibold uppercase tracking-widest text-muted">Admin</p>

        <nav className="mt-6 flex gap-1 overflow-x-auto lg:flex-col">
          {links.map((link) => {
            const active = pathname === link.href || pathname.startsWith(link.href + "/");
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`whitespace-nowrap rounded-xl px-3 py-2 text-sm font-semibold transition ${
                  active
                    ? "bg-brand-soft text-brand-dark"
                    : "text-slate-600 hover:bg-slate-100 hover:text-ink"
                }`}
              >
                {link.label}
              </Link>
            );
          })}
        </nav>
      </div>

      <button
        type="button"
        onClick={logout}
        disabled={leaving}
        className="rounded-xl border border-slate-200 px-3 py-2 text-left text-sm font-semibold text-slate-600 transition hover:bg-slate-50 disabled:opacity-50"
      >
        {leaving ? "Saindo..." : "Sair"}
      </button>
    </div>
  );
}
