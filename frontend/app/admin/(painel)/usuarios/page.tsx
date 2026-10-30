"use client";

import { useEffect, useState } from "react";
import { apiErrorMessage, apiGet, apiPatch } from "@/lib/admin/api";
import type { AdminUser, Page, UserFilters } from "@/lib/admin/types";
import { badgeClass, label, roleLabels, userStatusLabels } from "@/lib/admin/labels";
import { formatDate } from "@/lib/admin/format";
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
  secondaryButtonClass,
} from "@/components/admin/ui";

const emptyFilters: UserFilters = { q: "", role: "", status: "" };

export default function UsuariosPage() {
  const [filters, setFilters] = useState<UserFilters>(emptyFilters);
  const [applied, setApplied] = useState<UserFilters>(emptyFilters);
  const [page, setPage] = useState(1);
  const [refreshKey, setRefreshKey] = useState(0);
  const [data, setData] = useState<Page<AdminUser> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyID, setBusyID] = useState<string | null>(null);

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
        const result = await apiGet<Page<AdminUser>>(`/api/v1/admin/users?${params.toString()}`);
        if (active) setData(result);
      } catch (err) {
        if (active) setError(apiErrorMessage(err, "Erro ao carregar usuarios."));
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

  async function changeStatus(user: AdminUser, status: "active" | "blocked" | "inactive") {
    let reason = "";
    if (status !== "active") {
      const input = window.prompt(
        `Motivo para ${userStatusLabels[status].toLowerCase()} ${user.display_name}:`,
      );
      if (input === null) return;
      reason = input.trim();
      if (!reason) {
        setError("Informe o motivo da alteracao de status.");
        return;
      }
    }

    const warning =
      status === "inactive"
        ? "A conta perdera o acesso imediatamente, inclusive sessoes abertas."
        : status === "blocked"
          ? `A conta nao podera criar novos agendamentos. Ha ${user.future_appointments} agendamento(s) futuro(s).`
          : "A conta voltara a ter acesso normal.";
    if (!window.confirm(`${warning}\n\nConfirmar?`)) return;

    setBusyID(user.id);
    setError(null);
    try {
      await apiPatch(`/api/v1/admin/users/${user.id}/status`, { status, reason });
      setRefreshKey((key) => key + 1);
    } catch (err) {
      setError(apiErrorMessage(err, "Erro ao atualizar status do usuario."));
    } finally {
      setBusyID(null);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Usuarios"
        description="Consulte clientes e profissionais e gerencie bloqueios e inativacoes."
      />

      {error && <ErrorBanner message={error} onDismiss={() => setError(null)} />}

      <Card className="p-4">
        <form onSubmit={submit} className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <div className="xl:col-span-2">
            <label htmlFor="q" className={labelClass}>Buscar</label>
            <input id="q" className={inputClass} placeholder="Nome ou e-mail" value={filters.q} onChange={(event) => setFilters({ ...filters, q: event.target.value })} />
          </div>
          <div>
            <label htmlFor="role" className={labelClass}>Papel</label>
            <select id="role" className={inputClass} value={filters.role} onChange={(event) => setFilters({ ...filters, role: event.target.value })}>
              <option value="">Todos</option>
              <option value="client">Clientes</option>
              <option value="professional">Profissionais</option>
            </select>
          </div>
          <div>
            <label htmlFor="status" className={labelClass}>Status</label>
            <select id="status" className={inputClass} value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>
              <option value="">Todos</option>
              <option value="active">Ativos</option>
              <option value="blocked">Bloqueados</option>
              <option value="inactive">Inativos</option>
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
                <th className="px-4 py-3 font-semibold">Usuario</th>
                <th className="px-4 py-3 font-semibold">Papel</th>
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="px-4 py-3 font-semibold">Futuros</th>
                <th className="px-4 py-3 font-semibold">Cadastro</th>
                <th className="px-4 py-3 font-semibold">Acoes</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 align-top">
              <TableState loading={loading} empty={!data || data.items.length === 0} colSpan={6} />
              {!loading && data?.items.map((item) => (
                <tr key={item.id} className="hover:bg-slate-50/70">
                  <td className="px-4 py-3">
                    <div className="font-medium text-ink">{item.display_name}</div>
                    <div className="text-xs text-muted">{item.email}</div>
                    {(item.city || item.state) && <div className="text-xs text-muted">{[item.city, item.state].filter(Boolean).join(" / ")}</div>}
                  </td>
                  <td className="px-4 py-3 text-ink">{label(roleLabels, item.role)}</td>
                  <td className="px-4 py-3">
                    <StatusBadge label={label(userStatusLabels, item.status)} className={badgeClass(item.status)} />
                    {item.status_reason && <div className="mt-1 max-w-56 text-xs text-muted">{item.status_reason}</div>}
                  </td>
                  <td className="px-4 py-3 text-ink">{item.future_appointments}</td>
                  <td className="whitespace-nowrap px-4 py-3 text-ink">{formatDate(item.created_at)}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-2">
                      {item.status !== "active" && (
                        <button type="button" disabled={busyID === item.id} onClick={() => changeStatus(item, "active")} className={secondaryButtonClass}>
                          Reativar
                        </button>
                      )}
                      {item.status !== "blocked" && (
                        <button type="button" disabled={busyID === item.id} onClick={() => changeStatus(item, "blocked")} className={secondaryButtonClass}>
                          Bloquear
                        </button>
                      )}
                      {item.status !== "inactive" && (
                        <button type="button" disabled={busyID === item.id} onClick={() => changeStatus(item, "inactive")} className={dangerButtonClass}>
                          Inativar
                        </button>
                      )}
                    </div>
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
