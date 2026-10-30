import { apiGet } from "./api";
import type {
  ApiEnvelope,
  Catalogs,
  Neighborhood,
  ProfessionalProfile,
  Slot,
  Specialty,
} from "./types";

// Nome canônico da linha do catálogo de públicos que exige `audience_other`.
export const OTHER_AUDIENCE_NAME = "Outro público específico";

export async function listCatalogs(): Promise<Catalogs> {
  const res = await apiGet<ApiEnvelope<Catalogs>>("/api/v1/catalogs");
  return (
    res.data ?? { services: [], objectives: [], audiences: [], locations: [] }
  );
}

export async function listProfessionals(): Promise<ProfessionalProfile[]> {
  const res = await apiGet<ApiEnvelope<ProfessionalProfile[]>>(
    "/api/v1/professionals",
  );
  return res.data ?? [];
}

export async function getProfessional(
  id: string,
): Promise<ProfessionalProfile | null> {
  const res = await apiGet<ApiEnvelope<ProfessionalProfile>>(
    `/api/v1/professionals/${id}`,
  );
  return res.data ?? null;
}

export async function listSpecialties(): Promise<Specialty[]> {
  const res = await apiGet<ApiEnvelope<Specialty[]>>("/api/v1/specialties");
  return res.data ?? [];
}

export async function listNeighborhoods(
  city?: string,
  state?: string,
): Promise<Neighborhood[]> {
  const params = new URLSearchParams();
  if (city) params.set("city", city);
  if (state) params.set("state", state);
  const query = params.toString();
  const res = await apiGet<ApiEnvelope<Neighborhood[]>>(
    `/api/v1/neighborhoods${query ? `?${query}` : ""}`,
  );
  return res.data ?? [];
}

export async function getProfessionalSlots(
  id: string,
  date: string,
): Promise<Slot[]> {
  const res = await apiGet<ApiEnvelope<Slot[]>>(
    `/api/v1/professionals/${id}/slots?date=${encodeURIComponent(date)}`,
  );
  return res.data ?? [];
}
