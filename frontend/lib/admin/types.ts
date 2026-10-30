export interface ApiEnvelope<T> {
  success: boolean;
  data: T;
  message?: string;
}

export interface Page<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
}

export interface Party {
  id: string;
  name: string;
  email: string;
}

export interface AdminAppointment {
  id: string;
  client: Party;
  professional: Party;
  starts_at: string;
  ends_at: string;
  status: string;
  specialty_name?: string;
  location_type?: string;
  location_notes?: string;
  payment_id?: string;
  amount_cents?: number;
  payment_status?: string;
  cancelled_at?: string;
  cancel_reason?: string;
  created_at: string;
}

export interface AdminPayment {
  id: string;
  appointment_id: string;
  client: Party;
  professional: Party;
  amount_cents: number;
  platform_fee_cents: number;
  professional_net_cents: number;
  method: string;
  status: string;
  gateway_transaction_id?: string;
  paid_at?: string;
  created_at: string;
}

export interface PaymentSummary {
  count: number;
  amount_cents: number;
  platform_fee_cents: number;
  professional_net_cents: number;
  by_status: Record<string, number>;
}

export interface AdminUser {
  id: string;
  email: string;
  role: string;
  status: string;
  status_reason?: string;
  status_changed_at?: string;
  display_name: string;
  phone?: string;
  city?: string;
  state?: string;
  professional_id?: string;
  client_id?: string;
  created_at: string;
  updated_at: string;
  future_appointments: number;
}

export interface AdminAuthResponse {
  token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    role: string;
    status: string;
  };
}

export interface AppointmentFilters {
  client_q: string;
  professional_q: string;
  date_from: string;
  date_to: string;
  status: string;
}

export interface PaymentFilters {
  client_q: string;
  professional_q: string;
  date_from: string;
  date_to: string;
  date_field: string;
  status: string;
  method: string;
}

export interface UserFilters {
  q: string;
  role: string;
  status: string;
}
