"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { apiGet, apiPost } from "@/lib/api";
import type { ApiEnvelope } from "@/lib/types";

// "João da Silva" -> nome "João", iniciais "JS". "João" -> "J".
const PARTICLES = /^(de|da|do|das|dos)$/i;

function splitName(fullName: string): { firstName: string; initials: string } {
  const parts = fullName.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return { firstName: "", initials: "" };

  const firstName = parts[0];
  const surname = parts.slice(1).find((part) => !PARTICLES.test(part));
  const initials =
    firstName.charAt(0).toUpperCase() +
    (surname ? surname.charAt(0).toUpperCase() : "");

  return { firstName, initials };
}

export function Header() {
  const router = useRouter();
  const [firstName, setFirstName] = useState("");
  const [initials, setInitials] = useState("");

  useEffect(() => {
    let active = true;
    apiGet<ApiEnvelope<{ profile?: { name?: string } }>>("/api/v1/auth/me")
      .then((res) => {
        if (!active) return;
        const name = splitName(res.data?.profile?.name ?? "");
        setFirstName(name.firstName);
        setInitials(name.initials);
      })
      .catch(() => {
        // Sem nome, o cabeçalho fica com os campos vazios.
      });
    return () => {
      active = false;
    };
  }, []);

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
    <header className="sticky top-0 z-20 border-b border-slate-200 bg-surface/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center gap-6 px-4">
        <Link
          href="/app/client"
          className="flex items-center gap-2 text-lg font-extrabold text-ink"
        >
          <Image
            src="/images/logo-fitconecta.png"
            alt=""
            width={36}
            height={36}
            className="h-9 w-9 rounded-xl"
          />
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
          <Link
            href="/app/client/perfil"
            className="rounded-lg px-3 py-2 text-sm font-medium text-muted transition hover:bg-slate-100"
          >
            Meu perfil
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
              {initials}
            </div>
            <div className="hidden leading-tight sm:block">
              <p className="text-sm font-semibold text-ink">{firstName}</p>
              <p className="text-xs text-muted">Cliente</p>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}

export default Header;
