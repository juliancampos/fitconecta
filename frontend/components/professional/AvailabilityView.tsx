"use client";

import { useEffect, useState } from "react";
import { Card } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input } from "@/components/Input";
import { apiDelete, apiGet, apiPost } from "@/lib/api";
import { WEEKDAYS, formatDateTime, formatTime, weekdayLabel } from "@/lib/format";
import type {
  ApiEnvelope,
  AvailabilityException,
  AvailabilityRule,
} from "@/lib/types";

export function AvailabilityView() {
  const [rules, setRules] = useState<AvailabilityRule[]>([]);
  const [exceptions, setExceptions] = useState<AvailabilityException[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [savingRule, setSavingRule] = useState(false);
  const [savingException, setSavingException] = useState(false);

  const [weekday, setWeekday] = useState("1");
  const [startTime, setStartTime] = useState("08:00");
  const [endTime, setEndTime] = useState("18:00");
  const [slotMinutes, setSlotMinutes] = useState("60");

  const [excStart, setExcStart] = useState("");
  const [excEnd, setExcEnd] = useState("");
  const [excReason, setExcReason] = useState("");

  async function load() {
    const [r, e] = await Promise.all([
      apiGet<ApiEnvelope<AvailabilityRule[]>>("/api/v1/availability/rules"),
      apiGet<ApiEnvelope<AvailabilityException[]>>(
        "/api/v1/availability/exceptions",
      ),
    ]);
    setRules(r.data ?? []);
    setExceptions(e.data ?? []);
  }

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [r, e] = await Promise.all([
          apiGet<ApiEnvelope<AvailabilityRule[]>>("/api/v1/availability/rules"),
          apiGet<ApiEnvelope<AvailabilityException[]>>(
            "/api/v1/availability/exceptions",
          ),
        ]);
        if (!active) return;
        setRules(r.data ?? []);
        setExceptions(e.data ?? []);
        setError(null);
      } catch {
        if (active) {
          setError(
            "Não foi possível carregar sua disponibilidade. Esta área é restrita a profissionais.",
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

  async function handleAddRule(e: React.FormEvent) {
    e.preventDefault();
    setSavingRule(true);
    try {
      await apiPost("/api/v1/availability/rules", {
        weekday: Number(weekday),
        start_time: startTime,
        end_time: endTime,
        slot_minutes: Number(slotMinutes),
      });
      await load();
    } catch {
      setError("Não foi possível criar a regra. Verifique os horários.");
    } finally {
      setSavingRule(false);
    }
  }

  async function handleDeleteRule(id: string) {
    try {
      await apiDelete(`/api/v1/availability/rules/${id}`);
      setRules((prev) => prev.filter((rule) => rule.id !== id));
    } catch {
      setError("Não foi possível remover a regra.");
    }
  }

  async function handleAddException(e: React.FormEvent) {
    e.preventDefault();
    if (!excStart || !excEnd) return;
    setSavingException(true);
    try {
      await apiPost("/api/v1/availability/exceptions", {
        starts_at: new Date(excStart).toISOString(),
        ends_at: new Date(excEnd).toISOString(),
        reason: excReason,
      });
      setExcStart("");
      setExcEnd("");
      setExcReason("");
      await load();
    } catch {
      setError("Não foi possível adicionar o bloqueio. Verifique as datas.");
    } finally {
      setSavingException(false);
    }
  }

  async function handleDeleteException(id: string) {
    try {
      await apiDelete(`/api/v1/availability/exceptions/${id}`);
      setExceptions((prev) => prev.filter((exception) => exception.id !== id));
    } catch {
      setError("Não foi possível remover o bloqueio.");
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
        <h1 className="text-2xl font-bold text-ink">Disponibilidade</h1>
        <p className="mt-1 text-muted">
          Defina seus horários semanais e bloqueie datas específicas.
        </p>
      </div>

      {error && (
        <p className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600">
          {error}
        </p>
      )}

      <Card>
        <h2 className="text-lg font-semibold text-ink">
          Horários semanais
        </h2>

        <form
          onSubmit={handleAddRule}
          className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4 lg:items-end"
        >
          <div className="flex flex-col gap-1.5">
            <label htmlFor="weekday" className="text-sm font-medium text-ink">
              Dia da semana
            </label>
            <select
              id="weekday"
              value={weekday}
              onChange={(e) => setWeekday(e.target.value)}
              className="w-full rounded-xl border border-slate-300 bg-white px-3.5 py-2.5 text-sm text-ink outline-none focus:border-brand focus:ring-2 focus:ring-brand/20"
            >
              {WEEKDAYS.map((label, index) => (
                <option key={label} value={index}>
                  {label}
                </option>
              ))}
            </select>
          </div>

          <Input
            label="Início"
            name="start_time"
            type="time"
            value={startTime}
            onChange={(e) => setStartTime(e.target.value)}
          />
          <Input
            label="Fim"
            name="end_time"
            type="time"
            value={endTime}
            onChange={(e) => setEndTime(e.target.value)}
          />
          <Input
            label="Duração (min)"
            name="slot_minutes"
            type="number"
            min="15"
            step="15"
            value={slotMinutes}
            onChange={(e) => setSlotMinutes(e.target.value)}
          />

          <div className="sm:col-span-2 lg:col-span-4">
            <Button type="submit" disabled={savingRule}>
              {savingRule ? "Salvando..." : "Adicionar horário"}
            </Button>
          </div>
        </form>

        <div className="mt-6 flex flex-col gap-2">
          {rules.length === 0 ? (
            <p className="text-sm text-muted">
              Nenhum horário cadastrado ainda.
            </p>
          ) : (
            rules
              .slice()
              .sort((a, b) => a.weekday - b.weekday)
              .map((rule) => (
                <div
                  key={rule.id}
                  className="flex items-center justify-between rounded-xl border border-slate-200 px-4 py-3"
                >
                  <div className="text-sm">
                    <span className="font-semibold text-ink">
                      {weekdayLabel(rule.weekday)}
                    </span>
                    <span className="text-muted">
                      {" "}
                      · {formatTime(rule.start_time)}–{formatTime(rule.end_time)}{" "}
                      · {rule.slot_minutes} min
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleDeleteRule(rule.id)}
                    className="text-sm font-medium text-red-600 hover:underline"
                  >
                    Remover
                  </button>
                </div>
              ))
          )}
        </div>
      </Card>

      <Card className="mt-6">
        <h2 className="text-lg font-semibold text-ink">Bloqueios e exceções</h2>
        <p className="mt-1 text-sm text-muted">
          Bloqueie folgas, feriados ou compromissos pontuais.
        </p>

        <form
          onSubmit={handleAddException}
          className="mt-4 grid gap-4 sm:grid-cols-2"
        >
          <Input
            label="Início"
            name="exc_start"
            type="datetime-local"
            value={excStart}
            onChange={(e) => setExcStart(e.target.value)}
          />
          <Input
            label="Fim"
            name="exc_end"
            type="datetime-local"
            value={excEnd}
            onChange={(e) => setExcEnd(e.target.value)}
          />
          <Input
            label="Motivo (opcional)"
            name="exc_reason"
            placeholder="Ex.: feriado, viagem"
            value={excReason}
            onChange={(e) => setExcReason(e.target.value)}
          />
          <div className="flex items-end">
            <Button type="submit" disabled={savingException} variant="secondary">
              {savingException ? "Salvando..." : "Adicionar bloqueio"}
            </Button>
          </div>
        </form>

        <div className="mt-6 flex flex-col gap-2">
          {exceptions.length === 0 ? (
            <p className="text-sm text-muted">Nenhum bloqueio cadastrado.</p>
          ) : (
            exceptions.map((exception) => (
              <div
                key={exception.id}
                className="flex items-center justify-between rounded-xl border border-slate-200 px-4 py-3"
              >
                <div className="text-sm">
                  <span className="font-semibold text-ink">
                    {formatDateTime(exception.starts_at)} —{" "}
                    {formatDateTime(exception.ends_at)}
                  </span>
                  {exception.reason && (
                    <span className="text-muted"> · {exception.reason}</span>
                  )}
                </div>
                <button
                  type="button"
                  onClick={() => handleDeleteException(exception.id)}
                  className="text-sm font-medium text-red-600 hover:underline"
                >
                  Remover
                </button>
              </div>
            ))
          )}
        </div>
      </Card>
    </div>
  );
}

export default AvailabilityView;
