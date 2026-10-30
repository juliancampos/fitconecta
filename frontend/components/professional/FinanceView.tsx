"use client";

import { useEffect, useState } from "react";
import { Card } from "@/components/Card";
import { StatusBadge } from "@/components/professional/StatusBadge";
import { formatCurrency, formatDateTime } from "@/lib/format";
import { getMyAppointments } from "@/lib/professional";
import type { Appointment } from "@/lib/types";

export function FinanceView() {
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const apps = await getMyAppointments();
        if (active) setAppointments(apps);
      } catch {
        if (active) {
          setError(
            "Não foi possível carregar seus dados financeiros. Esta área é restrita a profissionais.",
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
      <div className="mx-auto max-w-5xl px-4 py-8">
        <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  const withPayment = appointments.filter((appointment) => appointment.payment);

  const received = withPayment
    .filter((appointment) => appointment.payment?.status === "paid")
    .reduce(
      (sum, appointment) =>
        sum + (appointment.payment?.professional_net_cents ?? 0),
      0,
    );

  const pending = withPayment
    .filter((appointment) => appointment.payment?.status === "pending")
    .reduce(
      (sum, appointment) =>
        sum + (appointment.payment?.professional_net_cents ?? 0),
      0,
    );

  const fees = withPayment
    .filter((appointment) => appointment.payment?.status === "paid")
    .reduce(
      (sum, appointment) =>
        sum + (appointment.payment?.platform_fee_cents ?? 0),
      0,
    );

  const stats = [
    { label: "Recebido (líquido)", value: formatCurrency(received) },
    { label: "A receber", value: formatCurrency(pending) },
    { label: "Taxas da plataforma", value: formatCurrency(fees) },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Financeiro</h1>
        <p className="mt-1 text-muted">
          Resumo dos seus repasses e pagamentos das sessões.
        </p>
      </div>

      {error && (
        <p className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600">
          {error}
        </p>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        {stats.map((stat) => (
          <Card key={stat.label} className="p-5">
            <p className="text-sm text-muted">{stat.label}</p>
            <p className="mt-1 text-2xl font-bold text-ink">{stat.value}</p>
          </Card>
        ))}
      </div>

      <h2 className="mt-8 text-lg font-semibold text-ink">Pagamentos</h2>

      {withPayment.length === 0 ? (
        <Card className="mt-3">
          <p className="text-sm text-muted">
            Nenhum registro financeiro por enquanto.
          </p>
        </Card>
      ) : (
        <div className="mt-3 flex flex-col gap-3">
          {withPayment.map((appointment) => (
            <Card
              key={appointment.id}
              className="flex flex-wrap items-center justify-between gap-4 p-4"
            >
              <div>
                <p className="font-semibold text-ink">
                  {appointment.client_name ?? "Cliente"}
                </p>
                <p className="text-sm text-muted">
                  {formatDateTime(appointment.starts_at)}
                </p>
              </div>
              <div className="flex items-center gap-6">
                <div className="text-right">
                  <p className="font-semibold text-ink">
                    {formatCurrency(
                      appointment.payment?.professional_net_cents ?? 0,
                    )}
                  </p>
                  <p className="text-xs text-muted">
                    de {formatCurrency(appointment.payment?.amount_cents ?? 0)}
                  </p>
                </div>
                <StatusBadge
                  status={
                    appointment.payment?.status === "paid"
                      ? "completed"
                      : "pending_payment"
                  }
                />
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

export default FinanceView;
