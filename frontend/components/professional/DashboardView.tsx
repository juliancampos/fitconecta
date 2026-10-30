"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { Card } from "@/components/Card";
import { StatusBadge } from "@/components/professional/StatusBadge";
import { formatCurrency, formatDateTime } from "@/lib/format";
import { getMyAppointments, getMyProfile } from "@/lib/professional";
import type { Appointment, ProfessionalProfile } from "@/lib/types";

export function DashboardView() {
  const [profile, setProfile] = useState<ProfessionalProfile | null>(null);
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [now, setNow] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [prof, apps] = await Promise.all([
          getMyProfile(),
          getMyAppointments(),
        ]);
        if (!active) return;
        setProfile(prof);
        setAppointments(apps);
        setNow(Date.now());
      } catch {
        if (active) {
          setError(
            "Não foi possível carregar seus dados. Esta área é restrita a profissionais.",
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

  if (loading) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-8">
        <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-8">
        <Card>
          <p className="text-muted">{error}</p>
        </Card>
      </div>
    );
  }

  const upcoming = appointments
    .filter(
      (appointment) =>
        new Date(appointment.starts_at).getTime() >= now &&
        (appointment.status === "confirmed" ||
          appointment.status === "pending_payment"),
    )
    .sort(
      (a, b) =>
        new Date(a.starts_at).getTime() - new Date(b.starts_at).getTime(),
    );

  const received = appointments
    .filter((appointment) => appointment.payment?.status === "paid")
    .reduce(
      (sum, appointment) =>
        sum + (appointment.payment?.professional_net_cents ?? 0),
      0,
    );

  const pending = appointments
    .filter((appointment) => appointment.payment?.status === "pending")
    .reduce(
      (sum, appointment) =>
        sum + (appointment.payment?.professional_net_cents ?? 0),
      0,
    );

  const stats = [
    { label: "Próximas sessões", value: String(upcoming.length) },
    { label: "A receber", value: formatCurrency(pending) },
    { label: "Já recebido", value: formatCurrency(received) },
    {
      label: "Especialidades",
      value: String(profile?.specialties?.length ?? 0),
    },
  ];

  const initial = (profile?.display_name ?? "P").charAt(0).toUpperCase();

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6 flex items-center gap-4">
        <div className="relative h-14 w-14 shrink-0 overflow-hidden rounded-full bg-brand-soft">
          {profile?.avatar_url ? (
            <Image
              src={profile.avatar_url}
              alt="Foto de perfil"
              width={56}
              height={56}
              className="h-full w-full object-cover"
            />
          ) : (
            <span className="flex h-full w-full items-center justify-center text-lg font-bold text-brand">
              {initial}
            </span>
          )}
        </div>
        <div>
          <h1 className="text-2xl font-bold text-ink sm:text-3xl">
            Olá, {profile?.display_name ?? "profissional"}!
          </h1>
          <p className="mt-1 text-muted">
            Este é o seu painel. Acompanhe sua agenda, disponibilidade e ganhos.
          </p>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.label} className="p-5">
            <p className="text-sm text-muted">{stat.label}</p>
            <p className="mt-1 text-2xl font-bold text-ink">{stat.value}</p>
          </Card>
        ))}
      </div>

      <div className="mt-8 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-ink">Próximas sessões</h2>
        <Link
          href="/app/professional/agendamentos"
          className="text-sm font-semibold text-brand hover:underline"
        >
          Ver todos
        </Link>
      </div>

      {upcoming.length === 0 ? (
        <Card className="mt-3">
          <p className="text-sm text-muted">
            Você ainda não possui sessões agendadas. Configure sua
            disponibilidade para começar a receber agendamentos.
          </p>
        </Card>
      ) : (
        <div className="mt-3 flex flex-col gap-3">
          {upcoming.slice(0, 5).map((appointment) => (
            <Card
              key={appointment.id}
              className="flex flex-wrap items-center justify-between gap-3 p-4"
            >
              <div>
                <p className="font-semibold text-ink">
                  {appointment.client_name ?? "Cliente"}
                </p>
                <p className="text-sm text-muted">
                  {formatDateTime(appointment.starts_at)}
                </p>
              </div>
              <div className="flex items-center gap-4">
                <span className="font-semibold text-ink">
                  {formatCurrency(
                    appointment.payment?.professional_net_cents ??
                      appointment.payment?.amount_cents ??
                      0,
                  )}
                </span>
                <StatusBadge status={appointment.status} />
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

export default DashboardView;
