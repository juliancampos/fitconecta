"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiGet, apiPost } from "@/lib/api";
import { Input } from "@/components/Input";
import { Button } from "@/components/Button";
import { NeighborhoodPicker } from "@/components/NeighborhoodPicker";
import type { ApiEnvelope, Neighborhood, Specialty } from "@/lib/types";

type Role = "client" | "professional";

interface FormState {
  role: Role;
  name: string;
  email: string;
  phone: string;
  birthdate: string;
  password: string;
  confirmPassword: string;
  cref: string;
  city: string;
  state: string;
  specialtyIds: number[];
  neighborhoods: Neighborhood[];
  sessionPrice: string;
  bio: string;
  acceptTerms: boolean;
}

type Errors = Partial<Record<keyof FormState | "form", string>>;

const initialState: FormState = {
  role: "client",
  name: "",
  email: "",
  phone: "",
  birthdate: "",
  password: "",
  confirmPassword: "",
  cref: "",
  city: "",
  state: "",
  specialtyIds: [],
  neighborhoods: [],
  sessionPrice: "",
  bio: "",
  acceptTerms: false,
};

function onlyDigits(value: string) {
  return value.replace(/\D/g, "");
}

function maskPhone(value: string) {
  const digits = onlyDigits(value).slice(0, 11);
  if (digits.length <= 2) return digits;
  if (digits.length <= 6) return `(${digits.slice(0, 2)}) ${digits.slice(2)}`;
  if (digits.length <= 10)
    return `(${digits.slice(0, 2)}) ${digits.slice(2, 6)}-${digits.slice(6)}`;
  return `(${digits.slice(0, 2)}) ${digits.slice(2, 7)}-${digits.slice(7)}`;
}

function validate(form: FormState): Errors {
  const errors: Errors = {};

  if (!form.name.trim()) errors.name = "Informe seu nome completo.";
  if (!form.email.trim()) {
    errors.email = "Informe seu e-mail.";
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    errors.email = "Informe um e-mail válido.";
  }
  if (onlyDigits(form.phone).length < 10) {
    errors.phone = "Informe um telefone com DDD.";
  }
  if (!form.birthdate) {
    errors.birthdate = "Informe sua data de nascimento.";
  } else if (new Date(form.birthdate) > new Date()) {
    errors.birthdate = "A data de nascimento não pode estar no futuro.";
  }
  if (form.password.length < 8) {
    errors.password = "A senha deve ter ao menos 8 caracteres.";
  }
  if (form.confirmPassword !== form.password) {
    errors.confirmPassword = "As senhas não conferem.";
  }
  if (form.role === "professional") {
    if (!form.cref.trim()) errors.cref = "Informe o número do CREF.";
    if (!form.city.trim()) errors.city = "Informe a cidade.";
    if (!form.state.trim()) errors.state = "Informe a UF.";
    if (form.specialtyIds.length === 0) {
      errors.specialtyIds = "Selecione ao menos uma especialidade.";
    }
    if (form.sessionPrice && Number(form.sessionPrice) <= 0) {
      errors.sessionPrice = "Informe um valor válido.";
    }
  }
  if (!form.acceptTerms) {
    errors.acceptTerms = "É necessário aceitar os termos de uso.";
  }

  return errors;
}

export default function RegisterPage() {
  const [form, setForm] = useState<FormState>(initialState);
  const [errors, setErrors] = useState<Errors>({});
  const [loading, setLoading] = useState(false);
  const [specialties, setSpecialties] = useState<Specialty[]>([]);
  const router = useRouter();

  useEffect(() => {
    let active = true;
    apiGet<ApiEnvelope<Specialty[]>>("/api/v1/specialties")
      .then((res) => {
        if (active && res.data) setSpecialties(res.data);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, []);

  function update<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function toggleSpecialty(id: number) {
    setForm((prev) => ({
      ...prev,
      specialtyIds: prev.specialtyIds.includes(id)
        ? prev.specialtyIds.filter((specialtyId) => specialtyId !== id)
        : [...prev.specialtyIds, id],
    }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const found = validate(form);
    setErrors(found);
    if (Object.keys(found).length > 0) return;

    setLoading(true);
    try {
      await apiPost("/api/v1/auth/register", {
        email: form.email,
        password: form.password,
        role: form.role,
        name: form.name,
        phone: onlyDigits(form.phone),
        birthdate: form.birthdate,
        cref_number: form.cref,
        city: form.city,
        state: form.state,
        specialty_ids:
          form.role === "professional" ? form.specialtyIds : undefined,
        neighborhood_ids:
          form.role === "professional"
            ? form.neighborhoods.map((n) => n.id)
            : undefined,
        bio: form.role === "professional" ? form.bio : undefined,
        session_price_cents:
          form.role === "professional" && form.sessionPrice
            ? Math.round(Number(form.sessionPrice) * 100)
            : undefined,
      });
      router.push("/auth/login");
    } catch {
      setErrors({ form: "Não foi possível concluir o cadastro. Tente novamente." });
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
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand text-white">
            🦜
          </span>
          FitConecta<span className="-ml-2 text-brand">+</span>
        </Link>
      </div>

      <div>
        <h2 className="text-2xl font-bold text-ink">Criar conta gratuita</h2>
        <p className="mt-1 text-sm text-muted">
          Todos os campos abaixo são obrigatórios.
        </p>
      </div>

      {errors.form && (
        <p
          role="alert"
          className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600"
        >
          {errors.form}
        </p>
      )}

      <div className="grid grid-cols-2 gap-2 rounded-xl bg-slate-100 p-1">
        {(
          [
            ["client", "Sou cliente"],
            ["professional", "Sou profissional"],
          ] as const
        ).map(([value, label]) => (
          <button
            key={value}
            type="button"
            onClick={() => update("role", value)}
            className={`rounded-lg py-2 text-sm font-semibold transition ${
              form.role === value
                ? "bg-white text-ink shadow-sm"
                : "text-muted hover:text-ink"
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <Input
          label="Nome completo"
          name="name"
          placeholder="Como você quer ser chamado"
          required
          value={form.name}
          onChange={(e) => update("name", e.target.value)}
          error={errors.name}
        />

        <Input
          label="E-mail"
          name="email"
          type="email"
          autoComplete="email"
          placeholder="voce@email.com"
          required
          value={form.email}
          onChange={(e) => update("email", e.target.value)}
          error={errors.email}
        />

        <div className="grid gap-4 sm:grid-cols-2">
          <Input
            label="Telefone"
            name="phone"
            type="tel"
            inputMode="numeric"
            placeholder="(31) 99999-9999"
            required
            value={form.phone}
            onChange={(e) => update("phone", maskPhone(e.target.value))}
            error={errors.phone}
          />

          <Input
            label="Data de nascimento"
            name="birthdate"
            type="date"
            required
            value={form.birthdate}
            onChange={(e) => update("birthdate", e.target.value)}
            error={errors.birthdate}
          />
        </div>

        {form.role === "professional" && (
          <>
            <Input
              label="Registro CREF"
              name="cref"
              placeholder="000000-G/UF"
              required
              value={form.cref}
              onChange={(e) => update("cref", e.target.value)}
              error={errors.cref}
            />

            <div className="grid gap-4 sm:grid-cols-[2fr_1fr]">
              <Input
                label="Cidade"
                name="city"
                placeholder="Belo Horizonte"
                required
                value={form.city}
                onChange={(e) => update("city", e.target.value)}
                error={errors.city}
              />
              <Input
                label="UF"
                name="state"
                placeholder="MG"
                maxLength={2}
                required
                value={form.state}
                onChange={(e) =>
                  update("state", e.target.value.toUpperCase())
                }
                error={errors.state}
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-sm font-medium text-ink">
                Especialidades<span className="text-brand"> *</span>
              </span>
              <div className="flex flex-wrap gap-2">
                {specialties.length === 0 && (
                  <span className="text-xs text-muted">
                    Carregando especialidades...
                  </span>
                )}
                {specialties.map((specialty) => {
                  const selected = form.specialtyIds.includes(specialty.id);
                  return (
                    <button
                      key={specialty.id}
                      type="button"
                      onClick={() => toggleSpecialty(specialty.id)}
                      aria-pressed={selected}
                      className={`rounded-full px-3 py-1.5 text-xs font-medium transition ${
                        selected
                          ? "bg-brand text-white"
                          : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                      }`}
                    >
                      {specialty.name}
                    </button>
                  );
                })}
              </div>
              {errors.specialtyIds && (
                <span className="text-xs font-medium text-red-600">
                  {errors.specialtyIds}
                </span>
              )}
            </div>

            <NeighborhoodPicker
              selected={form.neighborhoods}
              onChange={(list) => update("neighborhoods", list)}
              defaultCity={form.city}
              defaultState={form.state}
            />

            <Input
              label="Valor da sessão (R$)"
              name="sessionPrice"
              type="number"
              inputMode="decimal"
              min="1"
              step="1"
              placeholder="100"
              value={form.sessionPrice}
              onChange={(e) => update("sessionPrice", e.target.value)}
              error={errors.sessionPrice}
              hint="Se deixar em branco, usamos R$ 100,00 como valor padrão."
            />

            <div className="flex flex-col gap-1.5">
              <label htmlFor="bio" className="text-sm font-medium text-ink">
                Bio / apresentação
              </label>
              <textarea
                id="bio"
                name="bio"
                rows={3}
                placeholder="Conte sobre sua experiência, métodos e diferenciais."
                value={form.bio}
                onChange={(e) => update("bio", e.target.value)}
                className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20"
              />
            </div>
          </>
        )}

        <div className="grid gap-4 sm:grid-cols-2">
          <Input
            label="Senha"
            name="password"
            type="password"
            autoComplete="new-password"
            placeholder="Mínimo 8 caracteres"
            required
            value={form.password}
            onChange={(e) => update("password", e.target.value)}
            error={errors.password}
          />
          <Input
            label="Confirmar senha"
            name="confirmPassword"
            type="password"
            autoComplete="new-password"
            placeholder="Repita a senha"
            required
            value={form.confirmPassword}
            onChange={(e) => update("confirmPassword", e.target.value)}
            error={errors.confirmPassword}
          />
        </div>

        <label className="flex items-start gap-2 text-sm text-muted">
          <input
            type="checkbox"
            className="mt-0.5 accent-[var(--brand)]"
            checked={form.acceptTerms}
            onChange={(e) => update("acceptTerms", e.target.checked)}
          />
          <span>
            Li e aceito os{" "}
            <Link href="#" className="font-medium text-brand hover:underline">
              termos de uso
            </Link>{" "}
            e a{" "}
            <Link href="#" className="font-medium text-brand hover:underline">
              política de privacidade
            </Link>
            .
          </span>
        </label>
        {errors.acceptTerms && (
          <span className="-mt-2 text-xs font-medium text-red-600">
            {errors.acceptTerms}
          </span>
        )}

        <Button type="submit" fullWidth disabled={loading}>
          {loading ? "Criando conta..." : "Criar conta"}
        </Button>
      </form>

      <p className="text-center text-sm text-muted">
        Já tem conta?{" "}
        <Link
          href="/auth/login"
          className="font-semibold text-brand hover:underline"
        >
          Entrar
        </Link>
      </p>
    </div>
  );
}
