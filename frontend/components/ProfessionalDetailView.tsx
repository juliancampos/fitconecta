"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Image from "next/image";
import { Card } from "@/components/Card";
import { Calendar } from "@/components/Calendar";
import { Button } from "@/components/Button";
import { getProfessional, getProfessionalSlots, OTHER_AUDIENCE_NAME } from "@/lib/catalog";
import type { ProfessionalProfile, Slot } from "@/lib/types";

function todayKey(): string {
  const d = new Date();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

function initials(name: string) {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join("");
}

function ChipSection({
  title,
  items,
  brand = false,
}: {
  title: string;
  items: { id: number; name: string }[];
  brand?: boolean;
}) {
  if (items.length === 0) return null;
  return (
    <div className="mt-4">
      <h2 className="text-sm font-semibold text-ink">{title}</h2>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {items.map((item) => (
          <span
            key={item.id}
            className={`rounded-full px-2.5 py-1 text-xs font-medium ${
              brand ? "bg-brand-soft text-brand" : "bg-slate-100 text-slate-600"
            }`}
          >
            {item.name}
          </span>
        ))}
      </div>
    </div>
  );
}

export function ProfessionalDetailView() {
  const params = useParams<{ id: string }>();
  const router = useRouter();  const [professional, setProfessional] = useState<ProfessionalProfile | null>(
    null,
  );
  const [slotData, setSlotData] = useState<{
    key: string;
    slots: Slot[];
  } | null>(null);
  const [date, setDate] = useState("");
  const [loading, setLoading] = useState(true);

  const targetDate = date || todayKey();

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const profile = await getProfessional(params.id);
        if (active) setProfessional(profile);
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [params.id]);

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
  }, [professional, targetDate]);

  const loadedSlots = slotData?.key === targetDate;
  const availableSlots = loadedSlots ? slotData.slots : [];

  if (loading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-10">
        <div className="h-64 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  if (!professional) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-10">
        <Card>
          <p className="text-muted">Profissional não encontrado.</p>
        </Card>
      </div>
    );
  }

  const audiences = (professional.audiences ?? []).map((audience) =>
    audience.is_specific &&
    professional.audience_other &&
    audience.name === OTHER_AUDIENCE_NAME
      ? { ...audience, name: professional.audience_other }
      : audience,
  );

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <Card className="p-8">
        <div className="flex items-center gap-4">
          <div className="relative flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full bg-brand-soft text-xl font-bold text-brand">
            {professional.avatar_url ? (
              <Image
                src={professional.avatar_url}
                alt={professional.display_name}
                width={64}
                height={64}
                className="h-full w-full object-cover"
              />
            ) : (
              initials(professional.display_name)
            )}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-ink">
              {professional.display_name}
            </h1>
            <p className="text-sm text-muted">
              {professional.city}, {professional.state}
              {professional.reviews > 0 && (
                <>
                  {" · ★ "}
                  {professional.rating.toFixed(1)} ({professional.reviews}{" "}
                  avaliações)
                </>
              )}
            </p>
          </div>
        </div>

        <ChipSection
          title="Serviços oferecidos"
          items={professional.specialties ?? []}
        />

        <ChipSection
          title="Objetivos com os quais trabalha"
          items={professional.objectives ?? []}
        />

        <ChipSection title="Públicos atendidos" items={audiences} />

        {audiences.length > 0 && professional.audience_experience && (
          <p className="mt-2 text-sm text-muted">
            {professional.audience_experience}
          </p>
        )}

        <ChipSection
          title="Locais de atendimento presencial"
          items={professional.locations ?? []}
        />

        <ChipSection
          title="Bairros de atuação"
          items={professional.neighborhoods ?? []}
          brand
        />

        {professional.bio && (
          <p className="mt-4 text-sm text-muted">{professional.bio}</p>
        )}

        {professional.photos && professional.photos.length > 0 && (
          <div className="mt-6">
            <h2 className="text-lg font-semibold text-ink">Fotos</h2>
            <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3">
              {professional.photos.map((photo, index) => (
                <div
                  key={photo.id}
                  className="relative aspect-square overflow-hidden rounded-xl border border-slate-200 bg-slate-100"
                >
                  <Image
                    src={photo.url}
                    alt={`${professional.display_name} - foto ${index + 1}`}
                    fill
                    sizes="(max-width: 640px) 50vw, 33vw"
                    className="object-cover"
                  />
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="mt-6 flex items-center justify-between border-t border-slate-100 pt-4">
          <div>
            <span className="text-xl font-bold text-ink">
              R$ {(professional.session_price_cents / 100).toFixed(0)}
            </span>
            <span className="text-xs text-muted">/sessão</span>
          </div>
          <Button
            onClick={() =>
              router.push(`/app/client/agendamento/${professional.id}`)
            }
          >
            Agendar sessão
          </Button>
        </div>

        <h2 className="mt-8 text-lg font-semibold text-ink">
          Horários disponíveis
        </h2>

        <div className="mt-3 flex items-center gap-3">
          <input
            type="date"
            aria-label="Data"
            value={date}
            min={todayKey()}
            onChange={(e) => setDate(e.target.value)}
            className="rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          />
        </div>

        <div className="mt-3">
          {loadedSlots ? (
            <Calendar slots={availableSlots} />
          ) : (
            <div className="h-12 animate-pulse rounded-lg bg-slate-200" />
          )}
        </div>
      </Card>
    </div>
  );
}

export default ProfessionalDetailView;
