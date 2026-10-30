"use client";

import { useCallback, useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Card } from "@/components/Card";
import { Button } from "@/components/Button";
import { apiErrorMessage } from "@/lib/api";
import { createPixCharge, getAppointment } from "@/lib/appointments";
import { formatCurrency, formatDateTime, locationLabel } from "@/lib/format";
import type { Appointment, PixCharge } from "@/lib/types";

function formatCountdown(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export function PaymentView() {
  const params = useParams<{ appointmentId: string }>();
  const router = useRouter();

  const [appointment, setAppointment] = useState<Appointment | null>(null);
  const [charge, setCharge] = useState<PixCharge | null>(null);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [retryKey, setRetryKey] = useState(0);
  const [now, setNow] = useState(() => Date.now());

  const appointmentId = appointment?.id;
  const appointmentStatus = appointment?.status;

  const refreshAppointment = useCallback(async () => {
    const app = await getAppointment(params.appointmentId);
    setAppointment(app);
    return app;
  }, [params.appointmentId]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const app = await getAppointment(params.appointmentId);
        if (active) setAppointment(app);
      } catch (err) {
        if (active) {
          setError(
            apiErrorMessage(err, "Não foi possível carregar o agendamento."),
          );
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [params.appointmentId]);

  useEffect(() => {
    if (!appointmentId || appointmentStatus !== "pending_payment") return;
    let active = true;
    (async () => {
      setGenerating(true);
      setError(null);
      try {
        const result = await createPixCharge(appointmentId);
        if (active) setCharge(result);
      } catch (err) {
        if (active) {
          setError(
            apiErrorMessage(err, "Não foi possível gerar a cobrança Pix."),
          );
        }
      } finally {
        if (active) setGenerating(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [appointmentId, appointmentStatus, retryKey]);

  useEffect(() => {
    if (appointment?.status !== "pending_payment") return;
    let active = true;
    const interval = setInterval(async () => {
      try {
        const app = await getAppointment(params.appointmentId);
        if (!active) return;
        setAppointment(app);
        if (app?.status === "confirmed") setCharge(null);
      } catch {
        setError(null);
      }
    }, 4000);
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, [appointment?.status, params.appointmentId]);

  useEffect(() => {
    if (!charge) return;
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [charge]);

  async function handleCheck() {
    setChecking(true);
    setError(null);
    try {
      const app = await refreshAppointment();
      if (app?.status === "expired") setCharge(null);
    } catch (err) {
      setError(
        apiErrorMessage(err, "Não foi possível verificar o pagamento agora."),
      );
    } finally {
      setChecking(false);
    }
  }

  async function handleCopy() {
    if (!charge?.copy_paste) return;
    try {
      await navigator.clipboard.writeText(charge.copy_paste);
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      setError("Não foi possível copiar o código. Selecione e copie manualmente.");
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <div className="h-64 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  if (!appointment) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Card>
          <p className="text-muted">{error ?? "Agendamento não encontrado."}</p>
        </Card>
      </div>
    );
  }

  const amount = appointment.payment?.amount_cents ?? charge?.amount_cents ?? 0;

  if (appointment.status === "confirmed") {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Card className="p-6 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-2xl text-emerald-700">
            ✓
          </div>
          <h1 className="mt-4 text-2xl font-bold text-ink">
            Pagamento confirmado
          </h1>
          <p className="mt-1 text-muted">
            Seu horário com {appointment.professional_name ?? "o profissional"}{" "}
            está garantido na agenda.
          </p>
          <p className="mt-3 text-sm font-semibold text-ink">
            {formatDateTime(appointment.starts_at)}
          </p>
          <div className="mt-6">
            <Button onClick={() => router.push("/app/client/agendamentos")} fullWidth>
              Ver meus agendamentos
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  if (appointment.status === "expired") {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Card className="p-6 text-center">
          <h1 className="text-2xl font-bold text-ink">Reserva expirada</h1>
          <p className="mt-2 text-muted">
            O prazo de 10 minutos para o pagamento terminou e o horário foi
            liberado para outros clientes.
          </p>
          <div className="mt-6 flex flex-col gap-2">
            <Link href={`/app/client/agendamento/${appointment.professional_id}`}>
              <Button fullWidth>Escolher outro horário</Button>
            </Link>
            <Link href="/app/client/agendamentos">
              <Button variant="ghost" fullWidth>
                Ver meus agendamentos
              </Button>
            </Link>
          </div>
        </Card>
      </div>
    );
  }

  const expiresAt = charge?.expires_at ?? appointment.reserved_until;
  const secondsLeft = expiresAt
    ? Math.max(0, Math.floor((new Date(expiresAt).getTime() - now) / 1000))
    : null;
  const timeIsUp = secondsLeft !== null && secondsLeft <= 0;
  const hasCharge = Boolean(charge?.qr_code_image && charge?.copy_paste);

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <Card className="p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold text-ink">Pagamento via Pix</h1>
            <p className="mt-1 text-muted">
              Escaneie o QR Code ou use o Pix copia e cola. A confirmação é
              automática.
            </p>
          </div>
          <span className="inline-flex items-center gap-2 rounded-full bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-700">
            <span className="h-2 w-2 animate-pulse rounded-full bg-amber-500" />
            Aguardando pagamento
          </span>
        </div>

        <dl className="mt-6 space-y-3 text-sm">
          <div className="flex justify-between gap-4">
            <dt className="text-muted">Profissional</dt>
            <dd className="font-semibold text-ink">
              {appointment.professional_name ?? "Profissional"}
            </dd>
          </div>
          <div className="flex justify-between gap-4">
            <dt className="text-muted">Data e horário</dt>
            <dd className="text-right font-semibold text-ink">
              {formatDateTime(appointment.starts_at)} —{" "}
              {formatDateTime(appointment.ends_at)}
            </dd>
          </div>
          {appointment.specialty_name && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Tipo de aula</dt>
              <dd className="font-semibold text-ink">
                {appointment.specialty_name}
              </dd>
            </div>
          )}
          {appointment.location_type && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Local</dt>
              <dd className="font-semibold text-ink">
                {locationLabel(appointment.location_type)}
              </dd>
            </div>
          )}
          {appointment.location_notes && (
            <div className="flex justify-between gap-4">
              <dt className="text-muted">Descrição</dt>
              <dd className="max-w-[60%] text-right font-medium text-ink">
                {appointment.location_notes}
              </dd>
            </div>
          )}
          <div className="flex justify-between gap-4 border-t border-slate-100 pt-3">
            <dt className="font-semibold text-ink">Total</dt>
            <dd className="text-lg font-bold text-ink">
              {formatCurrency(amount)}
            </dd>
          </div>
        </dl>

        <div className="mt-6 rounded-2xl border border-slate-200 bg-slate-50 p-4">
          {generating && (
            <p className="text-sm font-medium text-muted">
              Gerando cobrança Pix...
            </p>
          )}

          {!generating && error && !hasCharge && (
            <div className="text-center">
              <p className="text-sm font-medium text-red-600">{error}</p>
              <div className="mt-3">
                <Button
                  variant="secondary"
                  onClick={() => setRetryKey((key) => key + 1)}
                >
                  Tentar novamente
                </Button>
              </div>
            </div>
          )}

          {!generating && hasCharge && (
            <div className="flex flex-col items-center">
              {!timeIsUp ? (
                <>
                  <p className="text-sm text-muted">
                    Reserve garantido por{" "}
                    <span className="font-mono text-base font-bold text-ink">
                      {formatCountdown(secondsLeft ?? 0)}
                    </span>
                  </p>
                  <div className="mt-3 rounded-xl bg-white p-3 shadow-sm">
                    <Image
                      src={charge!.qr_code_image!}
                      alt="QR Code para pagamento Pix"
                      width={240}
                      height={240}
                      unoptimized
                    />
                  </div>
                  <p className="mt-3 text-center text-xs text-muted">
                    Abra o app do seu banco, escolha Pix e escaneie o código.
                  </p>
                  <div className="mt-4 w-full">
                    <label
                      htmlFor="pix-copy-paste"
                      className="text-xs font-semibold uppercase tracking-wide text-muted"
                    >
                      Pix copia e cola
                    </label>
                    <textarea
                      id="pix-copy-paste"
                      readOnly
                      value={charge!.copy_paste}
                      rows={3}
                      className="mt-1 w-full resize-none rounded-xl border border-slate-300 bg-white px-3 py-2 font-mono text-xs text-ink"
                    />
                    <Button
                      variant="secondary"
                      onClick={handleCopy}
                      className="mt-2"
                      fullWidth
                    >
                      {copied ? "Código copiado" : "Copiar código Pix"}
                    </Button>
                  </div>
                </>
              ) : (
                <div className="py-2 text-center">
                  <p className="font-semibold text-ink">
                    O tempo da reserva terminou
                  </p>
                  <p className="mt-1 text-sm text-muted">
                    Se você pagou nos últimos minutos, a confirmação pode levar
                    até 2 minutos. Estamos verificando automaticamente.
                  </p>
                </div>
              )}
            </div>
          )}
        </div>

        {error && hasCharge && (
          <p className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-medium text-amber-700">
            {error}
          </p>
        )}

        <div className="mt-5 flex flex-col gap-2">
          <Button
            variant="secondary"
            onClick={handleCheck}
            disabled={checking}
            fullWidth
          >
            {checking ? "Verificando..." : "Já paguei, verificar agora"}
          </Button>
          <p className="text-center text-xs text-muted">
            A confirmação é automática. Você pode fechar esta página e
            acompanhar em Meus agendamentos.
          </p>
        </div>
      </Card>
    </div>
  );
}

export default PaymentView;
