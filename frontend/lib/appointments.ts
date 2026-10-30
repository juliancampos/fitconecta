import { apiGet, apiPost } from "./api";
import type {
  ApiEnvelope,
  Appointment,
  AppointmentOperationResult,
  CreateAppointmentInput,
  PixCharge,
} from "./types";

// Lista os agendamentos do usuario autenticado (cliente ou profissional).
export async function listMyAppointments(): Promise<Appointment[]> {
  const res = await apiGet<ApiEnvelope<Appointment[]>>("/api/v1/appointments");
  return res.data ?? [];
}

export async function getAppointment(id: string): Promise<Appointment | null> {
  const res = await apiGet<ApiEnvelope<Appointment>>(
    `/api/v1/appointments/${id}`,
  );
  return res.data ?? null;
}

// Cria o agendamento (pre-reserva de 10 min com status pending_payment).
export async function createAppointment(
  input: CreateAppointmentInput,
): Promise<AppointmentOperationResult> {
  const res = await apiPost<ApiEnvelope<AppointmentOperationResult>>(
    "/api/v1/appointments",
    {
      payment_method: "pix",
      ...input,
    },
  );
  if (!res.data) {
    throw new Error(res.message ?? "Nao foi possivel criar o agendamento.");
  }
  return res.data;
}

// Cria ou reutiliza a cobranca Pix da pre-reserva (ADR 0010).
export async function createPixCharge(id: string): Promise<PixCharge> {
  const res = await apiPost<ApiEnvelope<PixCharge>>(
    `/api/v1/appointments/${id}/payment`,
    {},
  );
  if (!res.data) {
    throw new Error(res.message ?? "Nao foi possivel gerar a cobranca Pix.");
  }
  return res.data;
}

export async function cancelAppointment(
  id: string,
  reason: string,
): Promise<Appointment> {
  const res = await apiPost<ApiEnvelope<Appointment>>(
    `/api/v1/appointments/${id}/cancel`,
    { reason },
  );
  if (!res.data) {
    throw new Error(res.message ?? "Nao foi possivel cancelar o agendamento.");
  }
  return res.data;
}
