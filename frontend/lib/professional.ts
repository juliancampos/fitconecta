import { apiDelete, apiGet, apiPost, apiPut } from "./api";
import type {
  ApiEnvelope,
  Appointment,
  Neighborhood,
  PresignResponse,
  ProfessionalPhoto,
  ProfessionalProfile,
} from "./types";

export const MAX_PHOTOS = 3;
export const MAX_NEIGHBORHOODS = 10;

export async function getMyProfile(): Promise<ProfessionalProfile | null> {
  const res = await apiGet<ApiEnvelope<ProfessionalProfile>>(
    "/api/v1/professionals/me/profile",
  );
  return res.data ?? null;
}

export async function getMyAppointments(): Promise<Appointment[]> {
  const res = await apiGet<ApiEnvelope<Appointment[]>>("/api/v1/appointments");
  return res.data ?? [];
}

// --- Midias (avatar e galeria) ---

// Solicita uma URL pre-assinada para upload direto ao storage (MinIO/R2).
export async function presignMedia(
  kind: "avatar" | "photo",
  contentType: string,
): Promise<PresignResponse> {
  const res = await apiPost<ApiEnvelope<PresignResponse>>(
    "/api/v1/professionals/me/media/presign",
    { kind, content_type: contentType },
  );
  if (!res.data) {
    throw new Error(res.message ?? "Nao foi possivel preparar o upload.");
  }
  return res.data;
}

// Envia o arquivo diretamente ao storage via PUT na URL pre-assinada.
// Nao usa cookies/credenciais: a autorizacao esta assinada na propria URL.
export async function uploadToStorage(
  uploadUrl: string,
  file: File,
): Promise<void> {
  const res = await fetch(uploadUrl, {
    method: "PUT",
    headers: { "Content-Type": file.type },
    body: file,
  });
  if (!res.ok) {
    throw new Error("Falha ao enviar a imagem para o armazenamento.");
  }
}

export async function setMyAvatar(
  avatarUrl: string,
): Promise<ProfessionalProfile | null> {
  const res = await apiPut<ApiEnvelope<ProfessionalProfile>>(
    "/api/v1/professionals/me/avatar",
    { avatar_url: avatarUrl },
  );
  return res.data ?? null;
}

export async function getMyPhotos(): Promise<ProfessionalPhoto[]> {
  const res = await apiGet<ApiEnvelope<ProfessionalPhoto[]>>(
    "/api/v1/professionals/me/photos",
  );
  return res.data ?? [];
}

export async function addMyPhoto(url: string): Promise<ProfessionalPhoto> {
  const res = await apiPost<ApiEnvelope<ProfessionalPhoto>>(
    "/api/v1/professionals/me/photos",
    { url },
  );
  if (!res.data) {
    throw new Error(res.message ?? "Nao foi possivel adicionar a foto.");
  }
  return res.data;
}

export async function deleteMyPhoto(id: string): Promise<void> {
  await apiDelete<ApiEnvelope<null>>(`/api/v1/professionals/me/photos/${id}`);
}

// --- Bairros de atuacao ---

export async function getMyNeighborhoods(): Promise<Neighborhood[]> {
  const res = await apiGet<ApiEnvelope<Neighborhood[]>>(
    "/api/v1/professionals/me/neighborhoods",
  );
  return res.data ?? [];
}

export async function setMyNeighborhoods(
  neighborhoodIds: number[],
): Promise<void> {
  await apiPut<ApiEnvelope<null>>(
    "/api/v1/professionals/me/neighborhoods",
    { neighborhood_ids: neighborhoodIds },
  );
}

// Fluxo completo: presign -> upload -> registra a URL.
export async function uploadImage(
  kind: "avatar" | "photo",
  file: File,
): Promise<string> {
  const presign = await presignMedia(kind, file.type);
  await uploadToStorage(presign.upload_url, file);
  return presign.public_url;
}
