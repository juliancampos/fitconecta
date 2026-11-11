"use client";

import { useEffect, useMemo, useState } from "react";
import { listNeighborhoods } from "@/lib/catalog";
import { MAX_NEIGHBORHOODS } from "@/lib/professional";
import type { Neighborhood } from "@/lib/types";

export function NeighborhoodPicker({
  selected,
  onChange,
  defaultCity = "",
  defaultState = "",
  max = MAX_NEIGHBORHOODS,
}: {
  selected: Neighborhood[];
  onChange: (list: Neighborhood[]) => void;
  defaultCity?: string;
  defaultState?: string;
  max?: number;
}) {
  const [city, setCity] = useState(defaultCity);
  const [state, setState] = useState(defaultState);
  const [options, setOptions] = useState<Neighborhood[]>([]);
  const [loading, setLoading] = useState(false);
  const [limitReached, setLimitReached] = useState(false);

  // Sincroniza o campo de busca quando a cidade/UF base mudam (sem efeito).
  const [syncedDefaults, setSyncedDefaults] = useState({
    city: defaultCity,
    state: defaultState,
  });
  if (
    defaultCity !== syncedDefaults.city ||
    defaultState !== syncedDefaults.state
  ) {
    setSyncedDefaults({ city: defaultCity, state: defaultState });
    setCity(defaultCity);
    setState(defaultState);
  }

  useEffect(() => {
    let active = true;
    const timer = setTimeout(async () => {
      setLoading(true);
      try {
        const list = await listNeighborhoods(city.trim(), state.trim());
        if (active) setOptions(list);
      } catch {
        if (active) setOptions([]);
      } finally {
        if (active) setLoading(false);
      }
    }, 400);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [city, state]);

  const selectedIds = useMemo(
    () => new Set(selected.map((n) => n.id)),
    [selected],
  );

  // Opcoes carregadas + selecionados que nao estao nas opcoes (outra cidade).
  const chips = useMemo(() => {
    const map = new Map<number, Neighborhood>();
    for (const n of options) map.set(n.id, n);
    for (const n of selected) if (!map.has(n.id)) map.set(n.id, n);
    return Array.from(map.values());
  }, [options, selected]);

  function toggle(neighborhood: Neighborhood) {
    if (selectedIds.has(neighborhood.id)) {
      setLimitReached(false);
      onChange(selected.filter((n) => n.id !== neighborhood.id));
      return;
    }
    if (selected.length >= max) {
      setLimitReached(true);
      return;
    }
    setLimitReached(false);
    onChange([...selected, neighborhood]);
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-ink">Bairros de atuação</span>
        <span className="text-xs font-semibold text-muted">
          {selected.length}/{max}
        </span>
      </div>
      <p className="text-xs text-muted">
        Selecione os bairros onde você atende. Você pode incluir bairros de mais
        de uma cidade.
      </p>

      <div className="grid gap-3 sm:grid-cols-[2fr_1fr]">
        <input
          aria-label="Cidade para buscar bairros"
          placeholder="Cidade"
          value={city}
          onChange={(e) => setCity(e.target.value)}
          className="w-full rounded-xl border border-slate-300 bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20"
        />
        <input
          aria-label="UF para buscar bairros"
          placeholder="UF"
          maxLength={2}
          value={state}
          onChange={(e) => setState(e.target.value.toUpperCase())}
          className="w-full rounded-xl border border-slate-300 bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition placeholder:text-slate-400 focus:border-brand focus:ring-2 focus:ring-brand/20"
        />
      </div>

      <div className="flex min-h-11 flex-wrap gap-2 rounded-xl border border-slate-200 bg-slate-50 p-2.5">
        {loading && chips.length === 0 && (
          <span className="px-1 py-1 text-xs text-muted">
            Carregando bairros...
          </span>
        )}
        {!loading && chips.length === 0 && (
          <span className="px-1 py-1 text-xs text-muted">
            {city.trim()
              ? "Nenhum bairro encontrado para esta cidade."
              : "Informe a cidade para ver os bairros disponíveis."}
          </span>
        )}
        {chips.map((neighborhood) => {
          const isSelected = selectedIds.has(neighborhood.id);
          return (
            <button
              key={neighborhood.id}
              type="button"
              onClick={() => toggle(neighborhood)}
              aria-pressed={isSelected}
              className={`rounded-full px-3 py-1.5 text-xs font-medium transition ${
                isSelected
                  ? "bg-brand text-white"
                  : "bg-surface text-slate-600 ring-1 ring-slate-200 hover:bg-slate-100"
              }`}
            >
              {neighborhood.name}
            </button>
          );
        })}
      </div>

      {limitReached && (
        <span className="text-xs font-medium text-red-600">
          Você pode selecionar no máximo {max} bairros.
        </span>
      )}
    </div>
  );
}

export default NeighborhoodPicker;
