"use client";

import type { Slot } from "@/lib/types";

interface CalendarProps {
  slots: Slot[];
  value?: string | null;
  onSelect?: (slot: Slot) => void;
  emptyText?: string;
}

function formatSlotTime(slot: Slot): string {
  return new Date(slot.starts_at).toLocaleTimeString("pt-BR", {
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function Calendar({ slots, value, onSelect, emptyText }: CalendarProps) {
  if (!slots || slots.length === 0) {
    return <div className="text-sm text-muted">{emptyText ?? "Sem disponibilidade."}</div>;
  }

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
      {slots.map((slot) => {
        const selected = value === slot.starts_at;

        if (!onSelect) {
          return (
            <div
              key={slot.starts_at}
              className="rounded-xl border border-slate-200 bg-surface px-3 py-2 text-center text-sm font-medium text-ink"
            >
              {formatSlotTime(slot)}
            </div>
          );
        }

        return (
          <button
            key={slot.starts_at}
            type="button"
            aria-pressed={selected}
            onClick={() => onSelect(slot)}
            className={`rounded-xl border px-3 py-2 text-sm font-semibold transition ${
              selected
                ? "border-brand bg-brand text-white"
                : "border-slate-200 bg-surface text-ink hover:border-brand hover:bg-brand-soft"
            }`}
          >
            {formatSlotTime(slot)}
          </button>
        );
      })}
    </div>
  );
}

export default Calendar;
