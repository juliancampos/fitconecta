export function formatCurrency(cents: number): string {
  return (cents / 100).toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
  });
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

export function formatTime(time: string): string {
  return time.slice(0, 5);
}

export const WEEKDAYS = [
  "Domingo",
  "Segunda",
  "Terça",
  "Quarta",
  "Quinta",
  "Sexta",
  "Sábado",
];

export function weekdayLabel(weekday: number): string {
  return WEEKDAYS[weekday] ?? String(weekday);
}

export const APPOINTMENT_STATUS_LABELS: Record<string, string> = {
  pending_payment: "Aguardando pagamento",
  confirmed: "Confirmado",
  cancelled_by_client: "Cancelado pelo cliente",
  cancelled_by_professional: "Cancelado pelo profissional",
  cancelled_by_admin: "Cancelado pelo admin",
  completed: "Concluído",
  no_show: "Não compareceu",
  expired: "Expirado",
};

export function statusLabel(status: string): string {
  return APPOINTMENT_STATUS_LABELS[status] ?? status;
}

export const LOCATION_TYPE_OPTIONS: { value: string; label: string }[] = [
  { value: "home", label: "Domicílio do cliente" },
  { value: "studio", label: "Estúdio do profissional" },
  { value: "online", label: "Online" },
];

export function locationLabel(location?: string | null): string {
  if (!location) return "";
  const found = LOCATION_TYPE_OPTIONS.find((option) => option.value === location);
  return found?.label ?? location;
}
