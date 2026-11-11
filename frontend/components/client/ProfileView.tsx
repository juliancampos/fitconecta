"use client";

import { useEffect, useState } from "react";
import { Card } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input } from "@/components/Input";
import { apiErrorMessage, apiGet, apiPut } from "@/lib/api";
import type {
  ApiEnvelope,
  ClientActivityLevel,
  ClientPersonalProfile,
  ClientSex,
} from "@/lib/types";

const SEX_OPTIONS: { value: ClientSex | ""; label: string }[] = [
  { value: "male", label: "Homem" },
  { value: "female", label: "Mulher" },
  { value: "other", label: "Outro" },
  { value: "", label: "Prefiro não informar" },
];

const ACTIVITY_OPTIONS: { value: ClientActivityLevel | ""; label: string }[] = [
  { value: "sedentary", label: "Sedentário" },
  { value: "casual", label: "Casualmente" },
  { value: "frequent", label: "Frequentemente" },
  { value: "", label: "Prefiro não informar" },
];

// /auth/me: só os campos exibidos no cartão "Conta".
interface MeResponse {
  user: { email: string };
  profile: { name: string; phone: string | null };
}

function todayISO(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

function formatBmi(value: number): string {
  return value.toLocaleString("pt-BR", {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  });
}

function onlyDigits(value: string): string {
  return value.replace(/\D/g, "").slice(0, 11);
}

// 52998224725 -> 529.982.247-25
function formatCPF(digits: string): string {
  return digits
    .replace(/^(\d{3})(\d)/, "$1.$2")
    .replace(/^(\d{3})\.(\d{3})(\d)/, "$1.$2.$3")
    .replace(/^(\d{3})\.(\d{3})\.(\d{3})(\d)/, "$1.$2.$3-$4");
}

function ChoiceGroup<T extends string>({
  legend,
  name,
  value,
  options,
  onChange,
}: {
  legend: string;
  name: string;
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
}) {
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm font-medium text-ink">{legend}</legend>
      <div className="flex flex-wrap gap-2">
        {options.map((option) => {
          const id = `${name}-${option.value || "none"}`;
          return (
            <label
              key={id}
              htmlFor={id}
              className="flex cursor-pointer items-center gap-2 rounded-xl border border-slate-300 px-3 py-2 text-sm text-ink transition hover:bg-slate-100 has-[:checked]:border-brand has-[:checked]:bg-brand-soft"
            >
              <input
                id={id}
                type="radio"
                name={name}
                value={option.value}
                checked={value === option.value}
                onChange={() => onChange(option.value)}
                className="h-4 w-4 accent-brand"
              />
              {option.label}
            </label>
          );
        })}
      </div>
    </fieldset>
  );
}

export function ProfileView() {
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [saving, setSaving] = useState(false);

  const [account, setAccount] = useState<MeResponse["profile"] | null>(null);
  const [email, setEmail] = useState("");
  const [summary, setSummary] = useState<{
    bmi: number | null;
    age: number | null;
  }>({ bmi: null, age: null });

  const [birthDate, setBirthDate] = useState("");
  const [cpf, setCpf] = useState("");
  const [sex, setSex] = useState<ClientSex | "">("");
  const [height, setHeight] = useState("");
  const [weight, setWeight] = useState("");
  const [activityLevel, setActivityLevel] = useState<ClientActivityLevel | "">(
    "",
  );
  const [objective, setObjective] = useState("");

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [profile, me] = await Promise.all([
          apiGet<ApiEnvelope<ClientPersonalProfile>>(
            "/api/v1/clients/me/profile",
          ),
          apiGet<ApiEnvelope<MeResponse>>("/api/v1/auth/me"),
        ]);
        if (!active) return;
        // Sem o perfil carregado o formulário salvaria tudo como null.
        if (!profile.data) throw new Error("Perfil vazio");
        const data = profile.data;
        setBirthDate(data.birth_date ?? "");
        setCpf(formatCPF(data.cpf ?? ""));
        setSex(data.sex ?? "");
        setHeight(String(data.height_cm ?? ""));
        setWeight(String(data.weight_kg ?? ""));
        setActivityLevel(data.activity_level ?? "");
        setObjective(data.objective ?? "");
        setSummary({ bmi: data.bmi, age: data.age });
        setAccount(me.data?.profile ?? null);
        setEmail(me.data?.user.email ?? "");
      } catch (err) {
        if (active) {
          setLoadError(
            apiErrorMessage(err, "Não foi possível carregar seus dados."),
          );
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaveError(null);
    setSuccess(false);
    setSaving(true);
    try {
      const res = await apiPut<ApiEnvelope<ClientPersonalProfile>>(
        "/api/v1/clients/me/profile",
        {
          birth_date: birthDate || null,
          sex: sex || null,
          height_cm: height ? Number(height) : null,
          weight_kg: weight ? Number(weight) : null,
          activity_level: activityLevel || null,
          objective: objective.trim() || null,
          cpf: onlyDigits(cpf) || null,
        },
      );
      setSummary({
        bmi: res.data?.bmi ?? null,
        age: res.data?.age ?? null,
      });
      setSuccess(true);
    } catch (err) {
      setSaveError(apiErrorMessage(err, "Não foi possível salvar seus dados."));
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-8">
        <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-8">
        <h1 className="mb-6 text-2xl font-bold text-ink">Meu perfil</h1>
        <p
          role="alert"
          className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600"
        >
          {loadError}
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Meu perfil</h1>
        <p className="mt-1 text-muted">
          Seus dados de contato e suas informações físicas.
        </p>
      </div>

      {saveError && (
        <p
          id="profile-error"
          role="alert"
          className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600"
        >
          {saveError}
        </p>
      )}
      {success && (
        <p
          role="status"
          className="mb-4 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-700"
        >
          Dados atualizados.
        </p>
      )}

      <form
        onSubmit={handleSubmit}
        className="flex flex-col gap-6"
        aria-describedby={saveError ? "profile-error" : undefined}
      >
        <Card>
          <h2 className="text-lg font-semibold text-ink">Conta</h2>
          <dl className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <dt className="text-xs font-medium text-muted">Nome</dt>
              <dd className="text-sm text-ink">
                {account?.name || "Não informado"}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-muted">E-mail</dt>
              <dd className="text-sm text-ink">{email || "Não informado"}</dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-muted">Telefone</dt>
              <dd className="text-sm text-ink">
                {account?.phone || "Não informado"}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-muted">Idade</dt>
              <dd className="text-sm text-ink">
                {summary.age !== null ? `${summary.age} anos` : "Não informado"}
              </dd>
            </div>
          </dl>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <Input
              label="Data de nascimento"
              name="birth_date"
              type="date"
              max={todayISO()}
              value={birthDate}
              onChange={(e) => setBirthDate(e.target.value)}
            />
            <Input
              label="CPF"
              name="cpf"
              type="text"
              inputMode="numeric"
              placeholder="000.000.000-00"
              maxLength={14}
              pattern="[0-9]{3}\.[0-9]{3}\.[0-9]{3}-[0-9]{2}"
              title="CPF no formato 000.000.000-00"
              value={cpf}
              onChange={(e) => setCpf(formatCPF(onlyDigits(e.target.value)))}
            />
          </div>
        </Card>

        <Card>
          <div className="mb-4 flex flex-wrap items-start justify-between gap-2">
            <div>
              <h2 className="text-lg font-semibold text-ink">Dados físicos</h2>
              <p className="mt-1 text-xs text-muted">
                Usados para personalizar seu treino.
              </p>
            </div>
            {summary.bmi !== null && (
              <span className="text-sm font-medium text-ink">
                IMC {formatBmi(summary.bmi)}
              </span>
            )}
          </div>

          <div className="flex flex-col gap-4">
            <ChoiceGroup
              legend="Sexo"
              name="sex"
              value={sex}
              options={SEX_OPTIONS}
              onChange={setSex}
            />

            <div className="grid gap-4 sm:grid-cols-2">
              <Input
                label="Altura (cm)"
                name="height_cm"
                type="number"
                min="100"
                max="250"
                inputMode="numeric"
                value={height}
                onChange={(e) => setHeight(e.target.value)}
              />
              <Input
                label="Peso (kg)"
                name="weight_kg"
                type="number"
                step="0.1"
                min="30"
                max="300"
                inputMode="decimal"
                value={weight}
                onChange={(e) => setWeight(e.target.value)}
              />
            </div>

            <ChoiceGroup
              legend="Faço atividades"
              name="activity_level"
              value={activityLevel}
              options={ACTIVITY_OPTIONS}
              onChange={setActivityLevel}
            />

            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="objective"
                className="text-sm font-medium text-ink"
              >
                Objetivo
              </label>
              <textarea
                id="objective"
                name="objective"
                rows={3}
                maxLength={500}
                placeholder="Ex.: correr 5 km sem parar, ganhar massa muscular..."
                value={objective}
                onChange={(e) => setObjective(e.target.value)}
                className="w-full rounded-xl border border-slate-300 bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20"
              />
            </div>

            <Button type="submit" disabled={saving}>
              {saving ? "Salvando..." : "Salvar alterações"}
            </Button>
          </div>
        </Card>
      </form>
    </div>
  );
}

export default ProfileView;
