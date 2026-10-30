export const appointmentStatusLabels: Record<string, string> = {
  pending_payment: "Aguardando pagamento",
  confirmed: "Confirmado",
  cancelled_by_client: "Cancelado pelo cliente",
  cancelled_by_professional: "Cancelado pelo profissional",
  cancelled_by_admin: "Cancelado pelo admin",
  completed: "Concluido",
  no_show: "Nao compareceu",
  expired: "Expirado",
};

export const paymentStatusLabels: Record<string, string> = {
  pending: "Pendente",
  paid: "Pago",
  failed: "Falhou",
  refund_pending: "Reembolso pendente",
  refunded: "Reembolsado",
  expired: "Expirado",
};

export const userStatusLabels: Record<string, string> = {
  active: "Ativo",
  blocked: "Bloqueado",
  inactive: "Inativo",
  pending: "Pendente",
};

export const roleLabels: Record<string, string> = {
  client: "Cliente",
  professional: "Profissional",
  admin: "Administrador",
};

export const methodLabels: Record<string, string> = {
  pix: "Pix",
  credit_card: "Cartao de credito",
};

export function label(map: Record<string, string>, value?: string): string {
  if (!value) return "-";
  return map[value] ?? value;
}

export function badgeClass(value?: string): string {
  switch (value) {
    case "active":
    case "confirmed":
    case "paid":
    case "completed":
      return "bg-emerald-50 text-emerald-700 ring-emerald-200";
    case "blocked":
    case "failed":
    case "no_show":
      return "bg-red-50 text-red-700 ring-red-200";
    case "inactive":
    case "expired":
    case "cancelled_by_client":
    case "cancelled_by_professional":
    case "cancelled_by_admin":
      return "bg-slate-100 text-slate-600 ring-slate-200";
    case "refund_pending":
    case "pending_payment":
      return "bg-amber-50 text-amber-700 ring-amber-200";
    default:
      return "bg-sky-50 text-sky-700 ring-sky-200";
  }
}
