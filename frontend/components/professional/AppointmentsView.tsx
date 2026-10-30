"use client";

import { useEffect, useState } from "react";
import { Card } from "@/components/Card";
import { StatusBadge } from "@/components/professional/StatusBadge";
import { apiPost } from "@/lib/api";
import { formatCurrency, formatDateTime } from "@/lib/format";
import { getMyAppointments } from "@/lib/professional";
import type { Appointment } from "@/lib/types";

const CANCELLABLE = ["pending_payment", "confirmed"];

export function AppointmentsView() {
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
            "Não foi possível carregar seus agendamentos. Esta área é restrita a profissionais.",
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

  async function handleCancel(id: string) {
    try {
      await apiPost(`/api/v1/appointments/${id}/cancel`, {
        reason: "Cancelado pelo profissional",
      });
      setAppointments((prev) =>
        prev.map((appointment) =>
          appointment.id === id
            ? { ...appointment, status: "cancelled_by_professional" }
            : appointment,
        ),
      );
    } catch {
      setError("Não foi possível cancelar o agendamento.");
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-8">
        <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Agendamentos</h1>
        <p className="mt-1 text-muted">
          Acompanhe as sessões agendadas com seus clientes.
        </p>
      </div>

      {error && (
        <p className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600">
          {error}
        </p>
      )}

      {appointments.length === 0 ? (
        <Card>
          <p className="text-sm text-muted">
            Nenhum agendamento por enquanto.
          </p>
        </Card>
      ) : (
        <div className="flex flex-col gap-3">
          {appointments.map((appointment) => (
            <Card
              key={appointment.id}
              className="flex flex-wrap items-center justify-between gap-4 p-4"
            >
              <div>
                <p className="font-semibold text-ink">
                  {appointment.client_name ?? "Cliente"}
                </p>
                <p className="text-sm text-muted">
                  {formatDateTime(appointment.starts_at)} —{" "}
                  {formatDateTime(appointment.ends_at)}
                </p>
              </div>
              <div className="flex items-center gap-4">
                <div className="text-right">
                  <p className="font-semibold text-ink">
                    {formatCurrency(
                      appointment.payment?.professional_net_cents ??
                        appointment.payment?.amount_cents ??
                        0,
                    )}
                  </p>
                  <p className="text-xs text-muted">seu repasse</p>
                </div>
                <StatusBadge status={appointment.status} />
                {CANCELLABLE.includes(appointment.status) && (
                  <button
                    type="button"
                    onClick={() => handleCancel(appointment.id)}
                    className="text-sm font-medium text-red-600 hover:underline"
                  >
                    Cancelar
                  </button>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

export default AppointmentsView;
