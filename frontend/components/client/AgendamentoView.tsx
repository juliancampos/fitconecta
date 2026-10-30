"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Card } from "@/components/Card";
import { Calendar } from "@/components/Calendar";
import { Button } from "@/components/Button";
import { apiErrorMessage } from "@/lib/api";
import { createAppointment } from "@/lib/appointments";
import { getProfessional, getProfessionalSlots } from "@/lib/catalog";
import { LOCATION_TYPE_OPTIONS, formatCurrency } from "@/lib/format";
import type { LocationType, ProfessionalProfile, Slot } from "@/lib/types";

function todayKey(): string {
  const d = new Date();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

export function AgendamentoView() {
  const params = useParams<{ professionalId: string }>();
  const router = useRouter();

  const [professional, setProfessional] = useState<ProfessionalProfile | null>(
    null,
  );
  const [date, setDate] = useState("");
  const [slotData, setSlotData] = useState<{ key: string; slots: Slot[] } | null>(
    null,
  );
  const [selectedSlot, setSelectedSlot] = useState<Slot | null>(null);
  const [specialtyId, setSpecialtyId] = useState<number | null>(null);
  const [locationType, setLocationType] = useState<LocationType | "">("");
  const [locationNotes, setLocationNotes] = useState("");
  const [reloadKey, setReloadKey] = useState(0);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const targetDate = date || todayKey();
  const specialties = professional?.specialties ?? [];
  const loadedSlots = slotData?.key === targetDate;
  const slots = loadedSlots ? slotData.slots : [];
  const locationStep = specialties.length > 0 ? 4 : 3;

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const profile = await getProfessional(params.professionalId);
        if (active) {
          setProfessional(profile);
          setSpecialtyId(profile?.specialties?.[0]?.id ?? null);
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [params.professionalId]);

  useEffect(() => {
    if (!professional) return;
    let active = true;
    (async () => {
      try {
        const result = await getProfessionalSlots(professional.id, targetDate);
        if (active) {
          setSlotData({
            key: targetDate,
            slots: result.filter((slot) => slot.available),
          });
        }
      } catch {
        if (active) setSlotData({ key: targetDate, slots: [] });
      }
    })();
    return () => {
      active = false;
    };
  }, [professional, targetDate, reloadKey]);

  async function handleConfirm() {
    if (!professional) return;
    setError(null);

    if (!selectedSlot) {
      setError("Escolha um horário disponível para continuar.");
      return;
    }
    if (specialties.length > 0 && specialtyId == null) {
      setError("Escolha o tipo de aula para continuar.");
      return;
    }
    if (!locationType) {
      setError("Escolha o local da sessão para continuar.");
      return;
    }

    setSubmitting(true);
    try {
      const result = await createAppointment({
        professional_id: professional.id,
        starts_at: selectedSlot.starts_at,
        ends_at: selectedSlot.ends_at,
        specialty_id: specialtyId ?? undefined,
        location_type: locationType,
        location_notes: locationNotes.trim() || undefined,
        payment_method: "pix",
      });
      router.push(`/app/client/pagamento/${result.appointment.id}`);
    } catch (err) {
      setError(
        apiErrorMessage(
          err,
          "Não foi possível concluir o agendamento. Tente novamente.",
        ),
      );
      // Recarrega os horários: o slot pode ter sido reservado por outro cliente.
      setSelectedSlot(null);
      setReloadKey((key) => key + 1);
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <div className="h-72 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  if (!professional) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <Card>
          <p className="text-muted">Profissional não encontrado.</p>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <button
        type="button"
        onClick={() => router.back()}
        className="mb-4 text-sm font-medium text-muted hover:text-ink"
      >
        ← Voltar
      </button>

      <Card className="p-6">
        <h1 className="text-2xl font-bold text-ink">Agendar sessão</h1>
        <p className="mt-1 text-muted">
          com <span className="font-semibold text-ink">{professional.display_name}</span>
          {" · "}
          {formatCurrency(professional.session_price_cents)}/sessão
        </p>

        <div className="mt-6">
          <h2 className="text-sm font-semibold text-ink">1. Escolha o dia</h2>
          <input
            type="date"
            aria-label="Dia da sessão"
            value={date}
            min={todayKey()}
            onChange={(e) => {
              setDate(e.target.value);
              setSelectedSlot(null);
            }}
            className="mt-2 rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          />
        </div>

        <div className="mt-6">
          <h2 className="text-sm font-semibold text-ink">2. Escolha o horário</h2>
          <div className="mt-2">
            {loadedSlots ? (
              <Calendar
                slots={slots}
                value={selectedSlot?.starts_at ?? null}
                onSelect={setSelectedSlot}
                emptyText="Nenhum horário disponível nesta data."
              />
            ) : (
              <div className="h-12 animate-pulse rounded-lg bg-slate-200" />
            )}
          </div>
        </div>

        {specialties.length > 0 && (
          <div className="mt-6">
            <h2 className="text-sm font-semibold text-ink">
              3. Tipo de aula
            </h2>
            <div className="mt-2 flex flex-wrap gap-2">
              {specialties.map((specialty) => {
                const active = specialtyId === specialty.id;
                return (
                  <button
                    key={specialty.id}
                    type="button"
                    aria-pressed={active}
                    onClick={() => setSpecialtyId(specialty.id)}
                    className={`rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                      active
                        ? "border-brand bg-brand text-white"
                        : "border-slate-200 bg-white text-ink hover:border-brand hover:bg-brand-soft"
                    }`}
                  >
                    {specialty.name}
                  </button>
                );
              })}
            </div>
          </div>
        )}

        <div className="mt-6">
          <h2 className="text-sm font-semibold text-ink">
            {locationStep}. Local da sessão
          </h2>
          <select
            aria-label="Local da sessão"
            value={locationType}
            onChange={(e) => setLocationType(e.target.value as LocationType | "")}
            className="mt-2 w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          >
            <option value="" disabled>
              Selecione o local...
            </option>
            {LOCATION_TYPE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>

          <label
            htmlFor="location-notes"
            className="mt-4 block text-sm font-semibold text-ink"
          >
            Descrição (opcional)
          </label>
          <textarea
            id="location-notes"
            value={locationNotes}
            onChange={(e) => setLocationNotes(e.target.value)}
            rows={3}
            maxLength={500}
            placeholder="Ex.: endereço completo, ponto de referência, observações..."
            className="mt-2 w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          />
        </div>

        {error && (
          <p className="mt-5 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600">
            {error}
          </p>
        )}
        <div className="mt-6 flex items-center justify-between border-t border-slate-100 pt-4">
          <div>
            <span className="text-xl font-bold text-ink">
              {formatCurrency(professional.session_price_cents)}
            </span>
            <span className="text-xs text-muted">/sessão</span>
            <p className="mt-1 text-xs text-muted">
              O horário fica reservado por 10 minutos até a confirmação do Pix.
            </p>
          </div>
          <Button onClick={handleConfirm} disabled={submitting}>
            {submitting ? "Reservando..." : "Continuar para pagamento"}
          </Button>
        </div>
      </Card>
    </div>
  );
}

export default AgendamentoView;
