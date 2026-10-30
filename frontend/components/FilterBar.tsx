"use client";

import type { Neighborhood } from "@/lib/types";

export interface Filters {
  query: string;
  specialty: string;
  neighborhood: string;
  date: string;
  time: string;
}

export const EMPTY_FILTERS: Filters = {
  query: "",
  specialty: "",
  neighborhood: "",
  date: "",
  time: "",
};

export interface TimeRange {
  value: string;
  label: string;
  start: number;
  end: number;
}

export const TIME_RANGES: TimeRange[] = [
  { value: "morning", label: "Manhã (06h–12h)", start: 6, end: 12 },
  { value: "afternoon", label: "Tarde (12h–18h)", start: 12, end: 18 },
  { value: "evening", label: "Noite (18h–22h)", start: 18, end: 22 },
];

export function FilterBar({
  value,
  onChange,
  onReset,
  resultCount,
  specialties,
  neighborhoods,
}: {
  value: Filters;
  onChange: (next: Filters) => void;
  onReset: () => void;
  resultCount: number;
  specialties: string[];
  neighborhoods: Neighborhood[];
}) {
  function set(patch: Partial<Filters>) {
    onChange({ ...value, ...patch });
  }

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm sm:p-5">
      <div className="grid gap-3">
        <div className="relative">
          <span
            className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400"
            aria-hidden
          >
            🔎
          </span>
          <input
            type="search"
            placeholder="Buscar por nome do profissional"
            value={value.query}
            onChange={(e) => set({ query: e.target.value })}
            className="w-full rounded-xl border border-slate-300 bg-white py-2.5 pl-10 pr-3.5 text-sm text-ink outline-none transition placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20"
          />
        </div>

        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <select
            aria-label="Serviço"
            value={value.specialty}
            onChange={(e) => set({ specialty: e.target.value })}
            className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          >
            <option value="">Todos os serviços</option>
            {specialties.map((specialty) => (
              <option key={specialty} value={specialty}>
                {specialty}
              </option>
            ))}
          </select>

          <select
            aria-label="Bairro"
            value={value.neighborhood}
            onChange={(e) => set({ neighborhood: e.target.value })}
            className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          >
            <option value="">Todos os bairros</option>
            {neighborhoods.map((neighborhood) => (
              <option key={neighborhood.id} value={String(neighborhood.id)}>
                {neighborhood.name} ({neighborhood.city}/{neighborhood.state})
              </option>
            ))}
          </select>

          <input
            type="date"
            aria-label="Data"
            value={value.date}
            onChange={(e) => set({ date: e.target.value })}
            className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          />

          <select
            aria-label="Horário"
            value={value.time}
            onChange={(e) => set({ time: e.target.value })}
            className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
          >
            <option value="">Qualquer horário</option>
            {TIME_RANGES.map((range) => (
              <option key={range.value} value={range.value}>
                {range.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between gap-3">
        <span className="text-sm text-muted">
          <strong className="text-ink">{resultCount}</strong> profissional(is)
          encontrado(s)
        </span>
        <button
          type="button"
          onClick={onReset}
          className="rounded-lg px-3 py-1.5 text-sm font-medium text-brand transition hover:bg-brand-soft"
        >
          Limpar filtros
        </button>
      </div>
    </section>
  );
}

export default FilterBar;
