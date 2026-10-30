import { statusLabel } from "@/lib/format";

const styles: Record<string, string> = {
  pending_payment: "bg-amber-50 text-amber-700",
  confirmed: "bg-emerald-50 text-emerald-700",
  cancelled_by_client: "bg-red-50 text-red-600",
  cancelled_by_professional: "bg-red-50 text-red-600",
  completed: "bg-slate-100 text-slate-600",
  no_show: "bg-slate-100 text-slate-600",
  expired: "bg-slate-100 text-slate-600",
};

export function StatusBadge({ status }: { status: string }) {
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${
        styles[status] ?? "bg-slate-100 text-slate-600"
      }`}
    >
      {statusLabel(status)}
    </span>
  );
}

export default StatusBadge;
