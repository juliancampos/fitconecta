"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { apiErrorMessage, apiGet, apiPost } from "@/lib/api";
import { Input } from "@/components/Input";
import { Button } from "@/components/Button";
import type { ApiEnvelope, ProfessionalBranch } from "@/lib/types";

type Role = "client" | "professional";

type CouncilCheck =
  | { status: "verified"; number: string; name: string }
  | { status: "pending"; message: string };

interface FormState {
  role: Role;
  name: string;
  email: string;
  phone: string;
  birthdate: string;
  password: string;
  confirmPassword: string;
  branch: ProfessionalBranch;
  cref: string;
  crn: string;
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
  branch: "physical_education",
  cref: "",
  crn: "",
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
    if (form.branch === "nutrition") {
      if (!form.crn.trim()) errors.crn = "Informe o número do CRN.";
    } else if (!form.cref.trim()) {
      errors.cref = "Informe o número do CREF.";
    }
  }
  if (!form.acceptTerms) {
    errors.acceptTerms = "É necessário aceitar os termos de uso.";
  }

  return errors;
}

function CouncilCheckStatus({
  council,
  checking,
  check,
}: {
  council: "cref" | "crn";
  checking: boolean;
  check: CouncilCheck | null;
}) {
  const label = council === "cref" ? "CREF" : "CRN";
  if (checking) {
    return (
      <span className="text-xs text-muted">
        {council === "cref"
          ? "Conferindo CREF no portal do CREF6..."
          : "Conferindo CRN na Consulta Nacional do CFN..."}
      </span>
    );
  }
  if (check?.status === "verified") {
    return (
      <span
        role="status"
        className="flex items-start gap-1.5 text-xs font-medium text-emerald-600"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="mt-px h-4 w-4 shrink-0"
        >
          <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.857-9.809a.75.75 0 0 0-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 1 0-1.06 1.061l2.5 2.5a.75.75 0 0 0 1.137-.089l4-5.5Z"
          />
        </svg>
        <span>
          {label} verificado: {check.number}
          {check.name ? ` — ${check.name}` : ""}
        </span>
      </span>
    );
  }
  if (check?.status === "pending") {
    return (
      <span
        role="status"
        className="flex items-start gap-1.5 text-xs font-medium text-amber-600"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="mt-px h-4 w-4 shrink-0"
        >
          <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm.75-13a.75.75 0 0 0-1.5 0v5c0 .414.336.75.75.75h4a.75.75 0 0 0 0-1.5h-3.25V5Z"
          />
        </svg>
        <span>
          {label} em análise. {check.message} O cadastro pode ser concluído
          normalmente.
        </span>
      </span>
    );
  }
  return null;
}

export default function RegisterPage() {
  const [form, setForm] = useState<FormState>(initialState);
  const [errors, setErrors] = useState<Errors>({});
  const [loading, setLoading] = useState(false);
  const [councilChecking, setCouncilChecking] = useState(false);
  const [crefCheck, setCrefCheck] = useState<CouncilCheck | null>(null);
  const [crnCheck, setCrnCheck] = useState<CouncilCheck | null>(null);
  const councilCheckId = useRef(0);
  const router = useRouter();

  function update<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  // Consulta informativa do conselho (CREF/CRN) no blur: nunca bloqueia o cadastro.
  async function checkCouncil(
    council: "cref" | "crn",
    value: string,
    setCheck: (check: CouncilCheck | null) => void,
  ) {
    const label = council === "cref" ? "CREF" : "CRN";
    const number = value.trim();
    const checkId = ++councilCheckId.current;
    if (!number) {
      setCouncilChecking(false);
      setCheck(null);
      return;
    }

    setCouncilChecking(true);
    try {
      const res = await apiGet<
        ApiEnvelope<{
          valid: boolean;
          message?: string;
          number?: string;
          name?: string;
        }>
      >(`/api/v1/professionals/${council}/validate?number=${encodeURIComponent(number)}`);
      if (checkId !== councilCheckId.current) return;
      const data = res.data;
      setCheck(
        data?.valid
          ? {
              status: "verified",
              number: data.number ?? number,
              name: data.name ?? "",
            }
          : {
              status: "pending",
              message: data?.message ?? `Não foi possível validar o ${label}.`,
            },
      );
    } catch {
      if (checkId !== councilCheckId.current) return;
      setCheck({
        status: "pending",
        message: `Não foi possível validar o ${label} agora.`,
      });
    } finally {
      if (checkId === councilCheckId.current) setCouncilChecking(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const found = validate(form);
    setErrors(found);
    if (Object.keys(found).length > 0) return;

    setLoading(true);
    try {
      const res = await apiPost<ApiEnvelope<{ user: { role: string } }>>("/api/v1/auth/register", {
        email: form.email,
        password: form.password,
        role: form.role,
        name: form.name,
        phone: onlyDigits(form.phone),
        birthdate: form.birthdate,
        branch: form.role === "professional" ? form.branch : undefined,
        cref_number:
          form.role === "professional" && form.branch === "physical_education"
            ? form.cref
            : undefined,
        crn_number:
          form.role === "professional" && form.branch === "nutrition"
            ? form.crn
            : undefined,
      });
      // Cadastro minimo: o profissional segue para o perfil completar os
      // dados (ADRs 0015 e 0016); o cliente vai direto para a vitrine.
      const role = res.data?.user.role;
      router.push(
        role === "professional" ? "/app/professional/perfil" : "/app/client",
      );
    } catch (error) {
      setErrors({
        form: apiErrorMessage(
          error,
          "Não foi possível concluir o cadastro. Tente novamente.",
        ),
      });
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
                ? "bg-surface text-ink shadow-sm"
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
            <div className="flex flex-col gap-1.5">
              <span className="text-sm font-medium text-ink">
                Ramo de atuação
              </span>
              <div
                role="group"
                aria-label="Ramo de atuação"
                className="grid grid-cols-2 gap-2 rounded-xl bg-slate-100 p-1"
              >
                {(
                  [
                    ["physical_education", "Educação Física"],
                    ["nutrition", "Nutrição"],
                  ] as const
                ).map(([value, label]) => (
                  <button
                    key={value}
                    type="button"
                    aria-pressed={form.branch === value}
                    onClick={() => {
                      update("branch", value);
                      // Descarta consultas em voo e limpa os selos do outro ramo.
                      councilCheckId.current++;
                      setCouncilChecking(false);
                      setCrefCheck(null);
                      setCrnCheck(null);
                    }}
                    className={`rounded-lg py-2 text-sm font-semibold transition ${
                      form.branch === value
                        ? "bg-surface text-ink shadow-sm"
                        : "text-muted hover:text-ink"
                    }`}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            {form.branch === "physical_education" && (
              <div className="flex flex-col gap-1.5">
                <Input
                  label="Registro CREF"
                  name="cref"
                  placeholder="000000-G/UF"
                  required
                  value={form.cref}
                  onChange={(e) => {
                    update("cref", e.target.value);
                    setCrefCheck(null);
                  }}
                  onBlur={(e) => checkCouncil("cref", e.target.value, setCrefCheck)}
                  error={errors.cref}
                />
                <CouncilCheckStatus
                  council="cref"
                  checking={councilChecking}
                  check={crefCheck}
                />
              </div>
            )}

            {form.branch === "nutrition" && (
              <div className="flex flex-col gap-1.5">
                <Input
                  label="Registro CRN"
                  name="crn"
                  placeholder="CRN9 12345 ou 12345"
                  required
                  value={form.crn}
                  onChange={(e) => {
                    update("crn", e.target.value);
                    setCrnCheck(null);
                  }}
                  onBlur={(e) => checkCouncil("crn", e.target.value, setCrnCheck)}
                  error={errors.crn}
                />
                <CouncilCheckStatus
                  council="crn"
                  checking={councilChecking}
                  check={crnCheck}
                />
              </div>
            )}

            <p className="text-sm text-muted">
              Cidade, especialidades, valor da sessão e demais dados você
              completa no seu perfil logo após o cadastro.
            </p>
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
