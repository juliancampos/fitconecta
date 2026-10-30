"use client";

import { useEffect, useState } from "react";
import { apiErrorMessage, apiGet } from "@/lib/admin/api";
import type { AdminPayment, Page, PaymentFilters, PaymentSummary } from "@/lib/admin/types";
import { badgeClass, label, methodLabels, paymentStatusLabels } from "@/lib/admin/labels";
import { formatCurrency, formatDateTime } from "@/lib/admin/format";
import {
  Card,
  ErrorBanner,
  PageHeader,
  Pagination,
  StatusBadge,
  TableState,
  inputClass,
  labelClass,
  primaryButtonClass,
} from "@/components/admin/ui";

const emptyFilters: PaymentFilters = {
  client_q: "",
  professional_q: "",
  date_from: "",
  date_to: "",
  date_field: "",
  status: "",
  method: "",
};

export default function PagamentosPage() {
  const [filters, setFilters] = useState<PaymentFilters>(emptyFilters);
  const [applied, setApplied] = useState<PaymentFilters>(emptyFilters);
  const [page, setPage] = useState(1);
  const [refreshKey, setRefreshKey] = useState(0);
  const [data, setData] = useState<Page<AdminPayment> | null>(null);
  const [summary, setSummary] = useState<PaymentSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    async function load() {
      setLoading(true);
      setError(null);
      try {
        const filterParams = new URLSearchParams();
        Object.entries(applied).forEach(([key, value]) => {
          if (value) filterParams.set(key, value);
        });
        const listParams = new URLSearchParams(filterParams);
        listParams.set("page", String(page));
        listParams.set("page_size", "20");

        const [list, totals] = await Promise.all([
          apiGet<Page<AdminPayment>>(`/api/v1/admin/payments?${listParams.toString()}`),
          apiGet<PaymentSummary>(`/api/v1/admin/payments/summary?${filterParams.toString()}`),
        ]);
        if (active) {
          setData(list);
          setSummary(totals);
        }
      } catch (err) {
        if (active) setError(apiErrorMessage(err, "Erro ao carregar pagamentos."));
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

  return (
    <div className="space-y-6">
      <PageHeader
        title="Pagamentos"
        description="Acompanhe valores, taxas, liquidos e reembolsos pendentes."
      />

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <SummaryCard title="Transacoes" value={String(summary?.count ?? 0)} />
        <SummaryCard title="Valor total" value={formatCurrency(summary?.amount_cents)} />
        <SummaryCard title="Taxa da plataforma" value={formatCurrency(summary?.platform_fee_cents)} />
        <SummaryCard title="Reembolsos pendentes" value={String(summary?.by_status?.refund_pending ?? 0)} accent />
      </div>

      <Card className="p-4">
        <form onSubmit={submit} className="grid gap-3 md:grid-cols-2 xl:grid-cols-8">
          <div>
            <label htmlFor="client_q" className={labelClass}>Cliente</label>
            <input id="client_q" className={inputClass} placeholder="Nome ou e-mail" value={filters.client_q} onChange={(event) => setFilters({ ...filters, client_q: event.target.value })} />
          </div>
          <div>
            <label htmlFor="professional_q" className={labelClass}>Profissional</label>
            <input id="professional_q" className={inputClass} placeholder="Nome ou e-mail" value={filters.professional_q} onChange={(event) => setFilters({ ...filters, professional_q: event.target.value })} />
          </div>
          <div>
            <label htmlFor="date_from" className={labelClass}>De</label>
            <input id="date_from" type="date" className={inputClass} value={filters.date_from} onChange={(event) => setFilters({ ...filters, date_from: event.target.value })} />
          </div>
          <div>
            <label htmlFor="date_to" className={labelClass}>Ate</label>
            <input id="date_to" type="date" className={inputClass} value={filters.date_to} onChange={(event) => setFilters({ ...filters, date_to: event.target.value })} />
          </div>
          <div>
            <label htmlFor="date_field" className={labelClass}>Data por</label>
            <select id="date_field" className={inputClass} value={filters.date_field} onChange={(event) => setFilters({ ...filters, date_field: event.target.value })}>
              <option value="">Criacao</option>
              <option value="paid_at">Pagamento</option>
            </select>
          </div>
          <div>
            <label htmlFor="status" className={labelClass}>Status</label>
            <select id="status" className={inputClass} value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>
              <option value="">Todos</option>
              {Object.entries(paymentStatusLabels).map(([value, text]) => (
                <option key={value} value={value}>{text}</option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="method" className={labelClass}>Metodo</label>
            <select id="method" className={inputClass} value={filters.method} onChange={(event) => setFilters({ ...filters, method: event.target.value })}>
              <option value="">Todos</option>
              <option value="pix">Pix</option>
              <option value="credit_card">Cartao</option>
            </select>
          </div>
          <div className="flex items-end gap-2">
            <button type="submit" className={`${primaryButtonClass} flex-1`}>Filtrar</button>
            <button type="button" onClick={clear} className="h-10 rounded-lg border border-slate-200 px-3 text-sm font-semibold text-slate-600">Limpar</button>
          </div>
        </form>
      </Card>

      <Card>
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="bg-slate-50 text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="px-4 py-3 font-semibold">Criado em</th>
                <th className="px-4 py-3 font-semibold">Cliente</th>
                <th className="px-4 py-3 font-semibold">Profissional</th>
                <th className="px-4 py-3 font-semibold">Valores</th>
                <th className="px-4 py-3 font-semibold">Metodo</th>
                <th className="px-4 py-3 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 align-top">
              <TableState loading={loading} empty={!data || data.items.length === 0} colSpan={6} />
              {!loading && data?.items.map((item) => (
                <tr key={item.id} className="hover:bg-slate-50/70">
                  <td className="whitespace-nowrap px-4 py-3 text-ink">
                    {formatDateTime(item.created_at)}
                    {item.paid_at && <div className="text-xs text-muted">Pago: {formatDateTime(item.paid_at)}</div>}
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
                    <div className="font-semibold text-ink">{formatCurrency(item.amount_cents)}</div>
                    <div className="text-xs text-muted">Taxa {formatCurrency(item.platform_fee_cents)}</div>
                    <div className="text-xs text-muted">Liquido {formatCurrency(item.professional_net_cents)}</div>
                  </td>
                  <td className="px-4 py-3 text-ink">{label(methodLabels, item.method)}</td>
                  <td className="px-4 py-3">
                    <StatusBadge label={label(paymentStatusLabels, item.status)} className={badgeClass(item.status)} />
                  </td>
                </tr>
              ))}
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

function SummaryCard({ title, value, accent }: { title: string; value: string; accent?: boolean }) {
  return (
    <Card className={`p-4 ${accent ? "border-amber-200 bg-amber-50" : ""}`}>
      <p className="text-xs font-semibold uppercase tracking-wide text-muted">{title}</p>
      <p className={`mt-1 text-2xl font-bold ${accent ? "text-amber-700" : "text-ink"}`}>{value}</p>
    </Card>
  );
}
