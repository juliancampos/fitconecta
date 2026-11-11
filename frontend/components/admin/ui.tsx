"use client";

import type { ReactNode } from "react";

export function StatusBadge({ label, className }: { label: string; className: string }) {
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ring-inset ${className}`}>
      {label}
    </span>
  );
}

export function ErrorBanner({ message, onDismiss }: { message: string; onDismiss?: () => void }) {
  return (
    <div role="alert" className="flex items-start justify-between gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-700">
      <span>{message}</span>
      {onDismiss && (
        <button type="button" onClick={onDismiss} className="text-red-500 hover:text-red-700" aria-label="Fechar erro">
          x
        </button>
      )}
    </div>
  );
}

export function PageHeader({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-ink">{title}</h1>
        {description && <p className="mt-1 text-sm text-muted">{description}</p>}
      </div>
      {children}
    </div>
  );
}

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-2xl border border-slate-200 bg-surface shadow-sm ${className}`}>{children}</div>;
}

export function Pagination({
  page,
  pageSize,
  total,
  onPageChange,
  disabled,
}: {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  disabled?: boolean;
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);
  return (
    <div className="flex flex-col gap-2 border-t border-slate-100 px-4 py-3 text-sm text-muted sm:flex-row sm:items-center sm:justify-between">
      <span>
        {from}-{to} de {total}
      </span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={disabled || page <= 1}
          className="rounded-lg border border-slate-200 bg-surface px-3 py-1.5 font-medium text-slate-700 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Anterior
        </button>
        <span className="px-1 font-medium text-slate-700">
          {page} / {totalPages}
        </span>
        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={disabled || page >= totalPages}
          className="rounded-lg border border-slate-200 bg-surface px-3 py-1.5 font-medium text-slate-700 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Proxima
        </button>
      </div>
    </div>
  );
}

export function TableState({ loading, empty, colSpan }: { loading: boolean; empty: boolean; colSpan: number }) {
  if (loading) {
    return (
      <tr>
        <td colSpan={colSpan} className="px-4 py-12 text-center text-sm text-muted">
          Carregando...
        </td>
      </tr>
    );
  }
  if (empty) {
    return (
      <tr>
        <td colSpan={colSpan} className="px-4 py-12 text-center text-sm text-muted">
          Nenhum registro encontrado.
        </td>
      </tr>
    );
  }
  return null;
}

export const inputClass =
  "h-10 w-full rounded-lg border border-slate-200 bg-surface px-3 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20";

export const labelClass = "mb-1 block text-xs font-semibold uppercase tracking-wide text-muted";

export const primaryButtonClass =
  "inline-flex h-10 items-center justify-center rounded-lg bg-brand px-4 text-sm font-semibold text-white transition hover:bg-brand-dark disabled:cursor-not-allowed disabled:opacity-60";

export const secondaryButtonClass =
  "inline-flex h-8 items-center justify-center rounded-lg border border-slate-200 bg-surface px-3 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40";

export const dangerButtonClass =
  "inline-flex h-8 items-center justify-center rounded-lg border border-red-200 bg-red-50 px-3 text-xs font-semibold text-red-700 transition hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-40";
