export interface ApiEnvelope<T> {
  success: boolean;
  data?: T;
  message?: string;
}

export interface Specialty {
  id: number;
  name: string;
}

export interface Objective {
  id: number;
  name: string;
}

export interface Audience {
  id: number;
  name: string;
  is_specific: boolean;
}

export interface ServiceLocation {
  id: number;
  name: string;
}

export interface Catalogs {
  services: Specialty[];
  objectives: Objective[];
  audiences: Audience[];
  locations: ServiceLocation[];
}

export interface Neighborhood {
  id: number;
  name: string;
  city: string;
  state: string;
  slug?: string;
}

export type ProfessionalBranch = "physical_education" | "nutrition";

export interface ProfessionalProfile {
  id: string;
  user_id: string;
  display_name: string;
  bio?: string;
  city: string;
  state: string;
  timezone: string;
  session_price_cents: number;
  branch: ProfessionalBranch;
  cref_number?: string;
  cref_verified_at?: string;
  crn_number?: string;
  crn_verified_at?: string;
  avatar_url?: string;
  rating: number;
  reviews: number;
  approved_at?: string;
  specialties?: Specialty[];
  neighborhoods?: Neighborhood[];
  objectives?: Objective[];
  audiences?: Audience[];
  audience_other?: string | null;
  audience_experience?: string | null;
  locations?: ServiceLocation[];
  photos?: ProfessionalPhoto[];
}

export type ClientSex = "male" | "female" | "other";
export type ClientActivityLevel = "sedentary" | "casual" | "frequent";

export interface ClientPersonalProfile {
  birth_date: string | null;
  sex: ClientSex | null;
  height_cm: number | null;
  weight_kg: number | null;
  activity_level: ClientActivityLevel | null;
  objective: string | null;
  cpf: string | null;
  bmi: number | null;
  age: number | null;
}

export interface Slot {
  starts_at: string;
  ends_at: string;
  available: boolean;
}

export interface ProfessionalPhoto {
  id: string;
  professional_id: string;
  url: string;
  position: number;
  created_at: string;
}

export interface PresignResponse {
  key: string;
  upload_url: string;
  public_url: string;
  expires_in: number;
}

export interface AvailabilityRule {
  id: string;
  professional_id: string;
  weekday: number;
  start_time: string;
  end_time: string;
  slot_minutes: number;
  created_at: string;
}

export interface AvailabilityException {
  id: string;
  professional_id: string;
  starts_at: string;
  ends_at: string;
  reason?: string;
  created_at: string;
}

export interface Payment {
  id?: string;
  amount_cents: number;
  platform_fee_cents: number;
  professional_net_cents: number;
  method?: "pix" | "credit_card";
  status: string;
  provider?: string;
  provider_payment_id?: string;
  expires_at?: string;
  paid_at?: string;
}

export interface PixCharge {
  payment_id: string;
  appointment_id: string;
  method: "pix";
  status: string;
  amount_cents: number;
  expires_at: string;
  qr_code_image?: string;
  copy_paste?: string;
  provider_payment_id?: string;
}

export type LocationType = "home" | "studio" | "online";

export interface Appointment {
  id: string;
  professional_id: string;
  client_id: string;
  starts_at: string;
  ends_at: string;
  specialty_id?: number;
  location_type?: LocationType;
  location_notes?: string;
  status: string;
  reserved_until?: string;
  professional_name?: string;
  client_name?: string;
  specialty_name?: string;
  payment?: Payment;
}

export interface CreateAppointmentInput {
  professional_id: string;
  starts_at: string;
  ends_at: string;
  specialty_id?: number;
  location_type?: LocationType;
  location_notes?: string;
  payment_method?: "pix";
}

export interface AppointmentOperationResult {
  appointment: Appointment;
  payment: Payment;
  message: string;
}
