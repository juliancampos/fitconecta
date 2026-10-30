"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Card } from "@/components/Card";
import { Button } from "@/components/Button";
import { StatusBadge } from "@/components/professional/StatusBadge";
import { apiErrorMessage } from "@/lib/api";
import { cancelAppointment, listMyAppointments } from "@/lib/appointments";
import { formatCurrency, formatDateTime, locationLabel } from "@/lib/format";
import type { Appointment } from "@/lib/types";

const CANCELLABLE = ["pending_payment", "confirmed"];

export function ClientAppointmentsView() {
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const apps = await listMyAppointments();
        if (active) setAppointments(apps);
      } catch (err) {
        if (active) {
          setError(
            apiErrorMessage(err, "Não foi possível carregar seus agendamentos."),
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
      await cancelAppointment(id, "Cancelado pelo cliente");
      setAppointments((prev) =>
        prev.map((appointment) =>
          appointment.id === id
            ? { ...appointment, status: "cancelled_by_client" }
            : appointment,
        ),
      );
    } catch (err) {
      setError(apiErrorMessage(err, "Não foi possível cancelar o agendamento."));
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-8">
        <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Meus agendamentos</h1>
        <p className="mt-1 text-muted">
          Acompanhe todas as suas sessões agendadas.
        </p>
      </div>

      {error && (
        <p className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600">
          {error}
        </p>
      )}

      {appointments.length === 0 ? (
        <Card className="text-center">
          <p className="text-sm text-muted">
            Você ainda não tem agendamentos.
          </p>
          <div className="mt-4">
            <Link href="/app/client">
              <Button>Buscar profissionais</Button>
            </Link>
          </div>
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
                  {appointment.professional_name ?? "Profissional"}
                </p>
                <p className="text-sm text-muted">
                  {formatDateTime(appointment.starts_at)} —{" "}
                  {formatDateTime(appointment.ends_at)}
                </p>
                {appointment.specialty_name && (
                  <p className="mt-1 text-xs font-medium text-brand">
                    {appointment.specialty_name}
                  </p>
                )}
                {appointment.location_type && (
                  <p className="mt-0.5 text-xs text-muted">
                    {locationLabel(appointment.location_type)}
                    {appointment.location_notes
                      ? ` · ${appointment.location_notes}`
                      : ""}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-4">
                <div className="text-right">
                  <p className="font-semibold text-ink">
                    {formatCurrency(
                      appointment.payment?.amount_cents ?? 0,
                    )}
                  </p>
                  <p className="text-xs text-muted">valor da sessão</p>
                </div>
                <StatusBadge status={appointment.status} />
                {appointment.status === "pending_payment" && (
                  <Link
                    href={`/app/client/pagamento/${appointment.id}`}
                    className="text-sm font-semibold text-brand hover:underline"
                  >
                    Pagar agora
                  </Link>
                )}
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

export default ClientAppointmentsView;
