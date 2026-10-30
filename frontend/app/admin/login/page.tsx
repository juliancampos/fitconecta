"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { apiErrorMessage, apiPost } from "@/lib/admin/api";
import type { AdminAuthResponse } from "@/lib/admin/types";
import { inputClass, labelClass, primaryButtonClass } from "@/components/admin/ui";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiPost<AdminAuthResponse>("/api/v1/admin/auth/login", { email, password });
      router.replace("/admin/agendamentos");
      router.refresh();
    } catch (err) {
      setError(apiErrorMessage(err, "Nao foi possivel entrar no painel."));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-950 px-4">
      <div className="w-full max-w-md rounded-3xl border border-slate-800 bg-white p-6 shadow-2xl sm:p-8">
        <div className="flex items-center gap-2">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand text-white">🦜</span>
          <div>
            <p className="text-lg font-extrabold leading-none text-ink">
              FitConecta<span className="text-brand">+</span>
            </p>
            <p className="text-xs font-semibold uppercase tracking-widest text-muted">Painel administrativo</p>
          </div>
        </div>

        <div className="mt-8">
          <h1 className="text-2xl font-bold text-ink">Acesso restrito</h1>
          <p className="mt-1 text-sm text-muted">Entre com uma conta administrativa.</p>
        </div>

        {error && (
          <p role="alert" className="mt-5 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-700">
            {error}
          </p>
        )}

        <form onSubmit={handleSubmit} className="mt-6 space-y-4" noValidate>
          <div>
            <label htmlFor="email" className={labelClass}>E-mail</label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              className={inputClass}
              placeholder="admin@fitconecta.com.br"
            />
          </div>
          <div>
            <label htmlFor="password" className={labelClass}>Senha</label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              className={inputClass}
              placeholder="••••••••"
            />
          </div>
          <button type="submit" disabled={loading} className={`${primaryButtonClass} w-full`}>
            {loading ? "Entrando..." : "Entrar"}
          </button>
        </form>
      </div>
    </div>
  );
}
