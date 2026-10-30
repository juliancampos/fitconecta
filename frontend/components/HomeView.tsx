"use client";

import { useEffect, useMemo, useState } from "react";
import { Card } from "@/components/Card";
import {
  EMPTY_FILTERS,
  FilterBar,
  TIME_RANGES,
  type Filters,
} from "@/components/FilterBar";
import { ProfessionalCard } from "@/components/ProfessionalCard";
import {
  getProfessionalSlots,
  listNeighborhoods,
  listProfessionals,
  listSpecialties,
} from "@/lib/catalog";
import type { Neighborhood, ProfessionalProfile } from "@/lib/types";

function todayKey(): string {
  const d = new Date();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

export function HomeView() {
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [professionals, setProfessionals] = useState<ProfessionalProfile[]>([]);
  const [specialties, setSpecialties] = useState<string[]>([]);
  const [neighborhoods, setNeighborhoods] = useState<Neighborhood[]>([]);
  const [availability, setAvailability] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [pros, specs, neigh] = await Promise.all([
          listProfessionals(),
          listSpecialties(),
          listNeighborhoods(),
        ]);
        if (!active) return;
        setProfessionals(pros);
        setSpecialties(specs.map((s) => s.name));
        setNeighborhoods(neigh);
      } catch {
        if (active) {
          setError("Não foi possível carregar os profissionais. Tente novamente.");
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!filters.date && !filters.time) {
      return;
    }
    let active = true;
    const date = filters.date || todayKey();
    const range = TIME_RANGES.find((r) => r.value === filters.time);

    (async () => {
      const entries = await Promise.all(
        professionals.map(async (p) => {
          try {
            const slots = await getProfessionalSlots(p.id, date);
            const ok = slots.some((slot) => {
              if (!slot.available) return false;
              if (!range) return true;
              const hour = new Date(slot.starts_at).getHours();
              return hour >= range.start && hour < range.end;
            });
            return [p.id, ok] as const;
          } catch {
            return [p.id, false] as const;
          }
        }),
      );
      if (active) setAvailability(Object.fromEntries(entries));
    })();

    return () => {
      active = false;
    };
  }, [filters.date, filters.time, professionals]);

  const filtered = useMemo(() => {
    const query = filters.query.trim().toLowerCase();
    const usesAvailability = Boolean(filters.date || filters.time);

    return professionals.filter((professional) => {
      if (
        query &&
        !professional.display_name.toLowerCase().includes(query)
      ) {
        return false;
      }
      if (
        filters.specialty &&
        !(professional.specialties ?? []).some(
          (s) => s.name === filters.specialty,
        )
      ) {
        return false;
      }
      if (filters.neighborhood) {
        const neighborhoodId = Number(filters.neighborhood);
        if (
          !(professional.neighborhoods ?? []).some(
            (n) => n.id === neighborhoodId,
          )
        ) {
          return false;
        }
      }
      if (usesAvailability && !availability[professional.id]) {
        return false;
      }
      return true;
    });
  }, [professionals, filters, availability]);

  if (loading) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-8">
        <div className="h-8 w-64 animate-pulse rounded-lg bg-slate-200" />
        <div className="mt-6 h-28 animate-pulse rounded-2xl bg-slate-200" />
        <div className="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-56 animate-pulse rounded-2xl bg-slate-200" />
          ))}
        </div>
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

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-ink sm:text-3xl">
          Encontre seu personal trainer
        </h1>
        <p className="mt-1 text-muted">
          Filtre por nome, serviço, data e horário e agende na hora.
        </p>
      </div>

      <FilterBar
        value={filters}
        onChange={setFilters}
        onReset={() => setFilters(EMPTY_FILTERS)}
        resultCount={filtered.length}
        specialties={specialties}
        neighborhoods={neighborhoods}
      />

      {filtered.length > 0 ? (
        <div className="mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((professional) => (
            <ProfessionalCard
              key={professional.id}
              professional={professional}
            />
          ))}
        </div>
      ) : (
        <div className="mt-6 flex flex-col items-center justify-center gap-3 rounded-2xl border border-dashed border-slate-300 bg-white px-6 py-16 text-center">
          <span className="text-4xl" aria-hidden>
            🦜
          </span>
          <h2 className="text-lg font-semibold text-ink">
            Nenhum profissional encontrado
          </h2>
          <p className="max-w-sm text-sm text-muted">
            Tente ajustar os filtros de serviço, data ou horário para ver mais
            resultados.
          </p>
        </div>
      )}
    </div>
  );
}

export default HomeView;
