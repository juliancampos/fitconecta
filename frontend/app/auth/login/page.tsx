"use client";

import { useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { apiPost } from "@/lib/api";
import { Input } from "@/components/Input";
import { Button } from "@/components/Button";
import type { ApiEnvelope } from "@/lib/types";

interface LoginData {
  user: { role: string };
}

// Traduz o erro da API/fetch para uma mensagem util ao usuario, em vez de
// sempre culpar as credenciais (pode ser falha de rede/CORS ou 401 real).
function describeLoginError(error: unknown): string {
  const raw = error instanceof Error ? error.message : "";
  try {
    const parsed = JSON.parse(raw) as { error?: string };
    if (parsed?.error) return parsed.error;
  } catch {
    // Nao e JSON: provavelmente falha de rede/CORS do fetch.
  }
  if (/fetch|network|failed to load/i.test(raw)) {
    const base =
      typeof window !== "undefined" ? window.location.origin : "/api";
    return `Não foi possível conectar à API. Verifique se o serviço está no ar em ${base}.`;
  }
  return "Não foi possível entrar. Verifique suas credenciais.";
}

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (!email || !password) {
      setError("Preencha o e-mail e a senha para continuar.");
      return;
    }

    setLoading(true);
    try {
      // O backend responde com um cookie de sessão (JWT) via /auth/login.
      const res = await apiPost<ApiEnvelope<LoginData>>("/api/v1/auth/login", {
        email,
        password,
      });
      const role = res.data?.user.role;
      router.push(role === "professional" ? "/app/professional" : "/app/client");
    } catch (error) {
      setError(describeLoginError(error));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="lg:hidden">
        <Link
          href="/auth/login"
          className="inline-flex items-center gap-2 text-lg font-extrabold text-ink"
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
      </div>

      <div>
        <h2 className="text-2xl font-bold text-ink">Entrar na plataforma</h2>
        <p className="mt-1 text-sm text-muted">
          Acesse com sua conta para buscar e agendar profissionais.
        </p>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600"
        >
          {error}
        </p>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <Input
          label="E-mail"
          name="email"
          type="email"
          autoComplete="email"
          placeholder="voce@email.com"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />

        <Input
          label="Senha"
          name="password"
          type="password"
          autoComplete="current-password"
          placeholder="••••••••"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        <div className="flex items-center justify-between text-sm">
          <label className="flex items-center gap-2 text-muted">
            <input type="checkbox" className="accent-[var(--brand)]" />
            Lembrar de mim
          </label>
          <Link
            href="#"
            className="font-medium text-brand hover:underline"
          >
            Esqueci minha senha
          </Link>
        </div>

        <Button type="submit" fullWidth disabled={loading}>
          {loading ? "Entrando..." : "Entrar"}
        </Button>
      </form>

      <p className="text-center text-sm text-muted">
        Ainda não tem conta?{" "}
        <Link
          href="/auth/register"
          className="font-semibold text-brand hover:underline"
        >
          Criar conta gratuita
        </Link>
      </p>
    </div>
  );
}
