"use client";

import { useEffect, useState } from "react";
import { apiErrorMessage, apiGet, apiPost } from "@/lib/admin/api";
import type { AdminAppointment, AppointmentFilters, Page } from "@/lib/admin/types";
import { appointmentStatusLabels, badgeClass, label, paymentStatusLabels } from "@/lib/admin/labels";
import { formatCurrency, formatDateTime } from "@/lib/admin/format";
import {
  Card,
  ErrorBanner,
  PageHeader,
  Pagination,
  StatusBadge,
  TableState,
  dangerButtonClass,
  inputClass,
  labelClass,
  primaryButtonClass,
} from "@/components/admin/ui";

const emptyFilters: AppointmentFilters = {
  client_q: "",
  professional_q: "",
  date_from: "",
  date_to: "",
  status: "",
};

export default function AgendamentosPage() {
  const [filters, setFilters] = useState<AppointmentFilters>(emptyFilters);
  const [applied, setApplied] = useState<AppointmentFilters>(emptyFilters);
  const [page, setPage] = useState(1);
  const [refreshKey, setRefreshKey] = useState(0);
  const [data, setData] = useState<Page<AdminAppointment> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionID, setActionID] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    async function load() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams();
        Object.entries(applied).forEach(([key, value]) => {
          if (value) params.set(key, value);
        });
        params.set("page", String(page));
        params.set("page_size", "20");
        const result = await apiGet<Page<AdminAppointment>>(
          `/api/v1/admin/appointments?${params.toString()}`,
        );
        if (active) setData(result);
      } catch (err) {
        if (active) setError(apiErrorMessage(err, "Erro ao carregar agendamentos."));
      } finally {
        if (active) setLoading(false);
      }
    }
    load();
    return () => {
      active = false;
    };
  }, [applied, page, refreshKey]);

  function submit(event: React.FormEvent) {
    event.preventDefault();
    setPage(1);
    setApplied({ ...filters });
    setRefreshKey((key) => key + 1);
  }

  function clear() {
    setFilters(emptyFilters);
    setApplied(emptyFilters);
    setPage(1);
    setRefreshKey((key) => key + 1);
  }

  async function cancelAppointment(item: AdminAppointment) {
    const reason = window.prompt(
      `Motivo do cancelamento de ${item.client.name} com ${item.professional.name}:`,
    );
    if (reason === null) return;
    if (!reason.trim()) {
      setError("Informe o motivo do cancelamento.");
      return;
    }
    if (!window.confirm("Confirmar cancelamento? Pagamentos pagos serao marcados como reembolso pendente.")) {
      return;
    }

    setActionID(item.id);
    setError(null);
    try {
      await apiPost(`/api/v1/admin/appointments/${item.id}/cancel`, { reason: reason.trim() });
      setRefreshKey((key) => key + 1);
    } catch (err) {
      setError(apiErrorMessage(err, "Erro ao cancelar agendamento."));
    } finally {
      setActionID(null);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Agendamentos"
        description="Filtre, acompanhe e cancele agendamentos da plataforma."
      />

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      <Card className="p-4">
        <form onSubmit={submit} className="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
          <div>
            <label htmlFor="client_q" className={labelClass}>Cliente</label>
            <input
              id="client_q"
              className={inputClass}
              placeholder="Nome ou e-mail"
              value={filters.client_q}
              onChange={(event) => setFilters({ ...filters, client_q: event.target.value })}
            />
          </div>
          <div>
            <label htmlFor="professional_q" className={labelClass}>Profissional</label>
            <input
              id="professional_q"
              className={inputClass}
              placeholder="Nome ou e-mail"
              value={filters.professional_q}
              onChange={(event) => setFilters({ ...filters, professional_q: event.target.value })}
            />
          </div>
          <div>
            <label htmlFor="date_from" className={labelClass}>De</label>
            <input
              id="date_from"
              type="date"
              className={inputClass}
              value={filters.date_from}
              onChange={(event) => setFilters({ ...filters, date_from: event.target.value })}
            />
          </div>
          <div>
            <label htmlFor="date_to" className={labelClass}>Ate</label>
            <input
              id="date_to"
              type="date"
              className={inputClass}
              value={filters.date_to}
              onChange={(event) => setFilters({ ...filters, date_to: event.target.value })}
            />
          </div>
          <div>
            <label htmlFor="status" className={labelClass}>Status</label>
            <select
              id="status"
              className={inputClass}
              value={filters.status}
              onChange={(event) => setFilters({ ...filters, status: event.target.value })}
            >
              <option value="">Todos</option>
              {Object.entries(appointmentStatusLabels).map(([value, text]) => (
                <option key={value} value={value}>{text}</option>
              ))}
            </select>
          </div>
          <div className="flex items-end gap-2">
            <button type="submit" className={`${primaryButtonClass} flex-1`}>Filtrar</button>
            <button type="button" onClick={clear} className="h-10 rounded-lg border border-slate-200 px-3 text-sm font-semibold text-slate-600">
              Limpar
            </button>
          </div>
        </form>
      </Card>

      <Card>
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="px-4 py-3 font-semibold">Data</th>
                <th className="px-4 py-3 font-semibold">Cliente</th>
                <th className="px-4 py-3 font-semibold">Profissional</th>
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="px-4 py-3 font-semibold">Pagamento</th>
                <th className="px-4 py-3 font-semibold">Acao</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 align-top">
              <TableState loading={loading} empty={!data || data.items.length === 0} colSpan={6} />
              {!loading && data?.items.map((item) => {
                const canCancel = item.status === "pending_payment" || item.status === "confirmed";
                return (
                  <tr key={item.id} className="hover:bg-slate-50/70">
                    <td className="whitespace-nowrap px-4 py-3 font-medium text-ink">
                      {formatDateTime(item.starts_at)}
                      {item.specialty_name && <div className="text-xs font-normal text-muted">{item.specialty_name}</div>}
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-ink">{item.client.name}</div>
                      <div className="text-xs text-muted">{item.client.email}</div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-ink">{item.professional.name}</div>
                      <div className="text-xs text-muted">{item.professional.email}</div>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge label={label(appointmentStatusLabels, item.status)} className={badgeClass(item.status)} />
                      {item.cancel_reason && <div className="mt-1 max-w-56 text-xs text-muted">{item.cancel_reason}</div>}
                    </td>
                    <td className="px-4 py-3">
                      {item.payment_status ? (
                        <div className="space-y-1">
                          <StatusBadge label={label(paymentStatusLabels, item.payment_status)} className={badgeClass(item.payment_status)} />
                          <div className="text-xs text-muted">{formatCurrency(item.amount_cents)}</div>
                        </div>
                      ) : (
                        <span className="text-xs text-muted">Sem pagamento</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {canCancel ? (
                        <button
                          type="button"
                          onClick={() => cancelAppointment(item)}
                          disabled={actionID === item.id}
                          className={dangerButtonClass}
                        >
                          {actionID === item.id ? "Cancelando..." : "Cancelar"}
                        </button>
                      ) : (
                        <span className="text-xs text-muted">-</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <Pagination
          page={data?.page ?? page}
          pageSize={data?.page_size ?? 20}
          total={data?.total ?? 0}
          onPageChange={setPage}
          disabled={loading}
        />
      </Card>
    </div>
  );
}
