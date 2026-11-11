"use client";

import { useEffect, useRef, useState } from "react";
import Image from "next/image";
import { Card } from "@/components/Card";
import { Button } from "@/components/Button";
import { Input } from "@/components/Input";
import { NeighborhoodPicker } from "@/components/NeighborhoodPicker";
import { apiErrorMessage, apiPut } from "@/lib/api";
import { OTHER_AUDIENCE_NAME, listCatalogs } from "@/lib/catalog";
import {
  MAX_PHOTOS,
  addMyPhoto,
  deleteMyPhoto,
  getMyProfile,
  setMyAvatar,
  setMyNeighborhoods,
  uploadImage,
} from "@/lib/professional";
import type {
  Catalogs,
  Neighborhood,
  ProfessionalBranch,
  ProfessionalPhoto,
} from "@/lib/types";

const ALLOWED_IMAGE_TYPES = [
  "image/jpeg",
  "image/png",
  "image/webp",
  "image/gif",
];
const MAX_IMAGE_BYTES = 5 * 1024 * 1024;

function validateImage(file: File): string | null {
  if (!ALLOWED_IMAGE_TYPES.includes(file.type)) {
    return "Formato inválido. Use JPEG, PNG, WEBP ou GIF.";
  }
  if (file.size > MAX_IMAGE_BYTES) {
    return "A imagem deve ter no máximo 5MB.";
  }
  return null;
}

function toggleId(ids: number[], id: number): number[] {
  return ids.includes(id) ? ids.filter((value) => value !== id) : [...ids, id];
}

function ChipToggle({
  label,
  selected,
  onClick,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={`rounded-full px-3 py-1.5 text-xs font-medium transition ${
        selected
          ? "bg-brand text-white"
          : "bg-slate-100 text-slate-600 hover:bg-slate-200"
      }`}
    >
      {label}
    </button>
  );
}

export function ProfileView() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const [displayName, setDisplayName] = useState("");
  const [branch, setBranch] = useState<ProfessionalBranch>("physical_education");
  const [cref, setCref] = useState("");
  const [crn, setCrn] = useState("");
  const [city, setCity] = useState("");
  const [state, setState] = useState("");
  const [sessionPrice, setSessionPrice] = useState("");
  const [bio, setBio] = useState("");

  const [catalogs, setCatalogs] = useState<Catalogs>({
    services: [],
    objectives: [],
    audiences: [],
    locations: [],
  });
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [objectiveIds, setObjectiveIds] = useState<number[]>([]);
  const [audienceIds, setAudienceIds] = useState<number[]>([]);
  const [audienceOther, setAudienceOther] = useState("");
  const [audienceExperience, setAudienceExperience] = useState("");
  const [showSpecificAudiences, setShowSpecificAudiences] = useState(false);
  const [locationIds, setLocationIds] = useState<number[]>([]);
  const [neighborhoods, setNeighborhoods] = useState<Neighborhood[]>([]);

  const [avatarUrl, setAvatarUrl] = useState<string | null>(null);
  const [photos, setPhotos] = useState<ProfessionalPhoto[]>([]);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [uploadingPhoto, setUploadingPhoto] = useState(false);
  const [mediaError, setMediaError] = useState<string | null>(null);

  const avatarInputRef = useRef<HTMLInputElement>(null);
  const photoInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const [profile, catalogData] = await Promise.all([
          getMyProfile(),
          listCatalogs(),
        ]);
        if (!active) return;
        setCatalogs(catalogData);
        if (profile) {
          setDisplayName(profile.display_name);
          setBranch(profile.branch ?? "physical_education");
          setCref(profile.cref_number ?? "");
          setCrn(profile.crn_number ?? "");
          setCity(profile.city);
          setState(profile.state);
          setSessionPrice(
            profile.session_price_cents
              ? String(profile.session_price_cents / 100)
              : "",
          );
          setBio(profile.bio ?? "");
          setSelectedIds((profile.specialties ?? []).map((s) => s.id));
          setObjectiveIds((profile.objectives ?? []).map((o) => o.id));
          const profileAudiences = profile.audiences ?? [];
          setAudienceIds(profileAudiences.map((a) => a.id));
          setAudienceOther(profile.audience_other ?? "");
          setAudienceExperience(profile.audience_experience ?? "");
          setShowSpecificAudiences(profileAudiences.some((a) => a.is_specific));
          setLocationIds((profile.locations ?? []).map((l) => l.id));
          setNeighborhoods(profile.neighborhoods ?? []);
          setAvatarUrl(profile.avatar_url ?? null);
          setPhotos(profile.photos ?? []);
        }
      } catch {
        if (active) {
          setError(
            "Não foi possível carregar seu perfil. Esta área é restrita a profissionais.",
          );
        }
      } finally {
        if (active) setLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  function toggleSpecialty(id: number) {
    setSelectedIds((prev) => toggleId(prev, id));
  }

  const otherAudienceId = catalogs.audiences.find(
    (a) => a.name === OTHER_AUDIENCE_NAME,
  )?.id;
  const otherAudienceSelected =
    otherAudienceId !== undefined && audienceIds.includes(otherAudienceId);
  const hasSpecificAudience = catalogs.audiences.some(
    (a) => a.is_specific && audienceIds.includes(a.id),
  );
  const specificAudienceIds = catalogs.audiences
    .filter((a) => a.is_specific)
    .map((a) => a.id);

  async function handleAvatarChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;

    const validation = validateImage(file);
    if (validation) {
      setMediaError(validation);
      return;
    }

    setMediaError(null);
    setSuccess(null);
    setUploadingAvatar(true);
    try {
      const publicUrl = await uploadImage("avatar", file);
      const updated = await setMyAvatar(publicUrl);
      setAvatarUrl(updated?.avatar_url ?? publicUrl);
      setSuccess("Foto de perfil atualizada.");
    } catch {
      setMediaError("Não foi possível enviar a foto de perfil.");
    } finally {
      setUploadingAvatar(false);
    }
  }

  async function handlePhotoAdd(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;

    const validation = validateImage(file);
    if (validation) {
      setMediaError(validation);
      return;
    }
    if (photos.length >= MAX_PHOTOS) {
      setMediaError(`Você pode ter no máximo ${MAX_PHOTOS} fotos.`);
      return;
    }

    setMediaError(null);
    setSuccess(null);
    setUploadingPhoto(true);
    try {
      const publicUrl = await uploadImage("photo", file);
      const photo = await addMyPhoto(publicUrl);
      setPhotos((prev) => [...prev, photo]);
    } catch {
      setMediaError("Não foi possível adicionar a foto.");
    } finally {
      setUploadingPhoto(false);
    }
  }

  async function handlePhotoRemove(id: string) {
    setMediaError(null);
    try {
      await deleteMyPhoto(id);
      setPhotos((prev) => prev.filter((photo) => photo.id !== id));
    } catch {
      setMediaError("Não foi possível remover a foto.");
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    if (otherAudienceSelected && !audienceOther.trim()) {
      setError(
        "Informe qual público você possui experiência em atender.",
      );
      return;
    }
    setSaving(true);
    try {
      await apiPut("/api/v1/professionals/me/profile", {
        display_name: displayName,
        bio,
        city,
        state,
        session_price_cents: Math.round(Number(sessionPrice || "0") * 100),
        cref_number: branch === "nutrition" ? "" : cref,
        crn_number: branch === "nutrition" ? crn : "",
      });
      await apiPut("/api/v1/professionals/me/specialties", {
        specialty_ids: selectedIds,
      });
      await setMyNeighborhoods(neighborhoods.map((n) => n.id));
      await apiPut("/api/v1/professionals/me/objectives", {
        objective_ids: objectiveIds,
      });
      await apiPut("/api/v1/professionals/me/audiences", {
        audience_ids: audienceIds,
        audience_other: otherAudienceSelected ? audienceOther.trim() : "",
        audience_experience: hasSpecificAudience ? audienceExperience : "",
      });
      await apiPut("/api/v1/professionals/me/locations", {
        location_ids: locationIds,
      });
      setSuccess("Perfil atualizado com sucesso.");
    } catch (error) {
      setError(
        apiErrorMessage(
          error,
          "Não foi possível salvar o perfil. Verifique os dados.",
        ),
      );
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-8">
        <div className="h-40 animate-pulse rounded-2xl bg-slate-200" />
      </div>
    );
  }

  const initial = (displayName || "P").charAt(0).toUpperCase();

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Perfil profissional</h1>
        <p className="mt-1 text-muted">
          Atualize suas informações públicas, foto e serviços oferecidos.
        </p>
      </div>

      {(error || mediaError) && (
        <p className="mb-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-600">
          {mediaError ?? error}
        </p>
      )}
      {success && (
        <p className="mb-4 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-700">
          {success}
        </p>
      )}

      <Card className="mb-6">
        <div className="flex flex-col items-start gap-5 sm:flex-row sm:items-center">
          <div className="relative h-24 w-24 shrink-0 overflow-hidden rounded-full bg-brand-soft">
            {avatarUrl ? (
              <Image
                src={avatarUrl}
                alt="Foto de perfil"
                width={96}
                height={96}
                className="h-full w-full object-cover"
              />
            ) : (
              <span className="flex h-full w-full items-center justify-center text-2xl font-bold text-brand">
                {initial}
              </span>
            )}
          </div>
          <div className="flex flex-col gap-2">
            <span className="text-sm font-medium text-ink">Foto de perfil</span>
            <p className="text-xs text-muted">
              JPEG, PNG, WEBP ou GIF. Tamanho máximo de 5MB.
            </p>
            <input
              ref={avatarInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleAvatarChange}
            />
            <Button
              type="button"
              variant="secondary"
              disabled={uploadingAvatar}
              onClick={() => avatarInputRef.current?.click()}
            >
              {uploadingAvatar ? "Enviando..." : "Alterar foto"}
            </Button>
          </div>
        </div>
      </Card>

      <Card className="mb-6">
        <div className="flex items-center justify-between">
          <div>
            <span className="text-sm font-medium text-ink">Fotos do perfil</span>
            <p className="text-xs text-muted">
              Até {MAX_PHOTOS} fotos exibidas no seu perfil público.
            </p>
          </div>
          <span className="text-xs font-semibold text-muted">
            {photos.length}/{MAX_PHOTOS}
          </span>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
          {photos.map((photo) => (
            <div
              key={photo.id}
              className="group relative aspect-square overflow-hidden rounded-xl border border-slate-200"
            >
              <Image
                src={photo.url}
                alt="Foto do perfil"
                fill
                sizes="(max-width: 640px) 50vw, 33vw"
                className="object-cover"
              />
              <button
                type="button"
                onClick={() => handlePhotoRemove(photo.id)}
                aria-label="Remover foto"
                className="absolute right-2 top-2 rounded-full bg-black/60 px-2 py-1 text-xs font-semibold text-white transition hover:bg-black/80"
              >
                Remover
              </button>
            </div>
          ))}

          {photos.length < MAX_PHOTOS && (
            <button
              type="button"
              onClick={() => photoInputRef.current?.click()}
              disabled={uploadingPhoto}
              className="flex aspect-square flex-col items-center justify-center gap-1 rounded-xl border-2 border-dashed border-slate-300 text-muted transition hover:border-brand hover:text-brand disabled:opacity-60"
            >
              <span className="text-2xl leading-none">+</span>
              <span className="text-xs font-medium">
                {uploadingPhoto ? "Enviando..." : "Adicionar"}
              </span>
            </button>
          )}
        </div>

        <input
          ref={photoInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={handlePhotoAdd}
        />
      </Card>

      <Card>
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Input
            label="Nome de exibição"
            name="display_name"
            required
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />

          {branch === "nutrition" ? (
            <Input
              label="Registro CRN"
              name="crn_number"
              placeholder="CRN9 12345 ou 12345"
              value={crn}
              onChange={(e) => setCrn(e.target.value)}
            />
          ) : (
            <Input
              label="Registro CREF"
              name="cref_number"
              placeholder="000000-G/UF"
              value={cref}
              onChange={(e) => setCref(e.target.value)}
            />
          )}

          <div className="grid gap-4 sm:grid-cols-[2fr_1fr]">
            <Input
              label="Cidade"
              name="city"
              value={city}
              onChange={(e) => setCity(e.target.value)}
            />
            <Input
              label="UF"
              name="state"
              maxLength={2}
              value={state}
              onChange={(e) => setState(e.target.value.toUpperCase())}
            />
          </div>

          <Input
            label="Valor da sessão (R$)"
            name="session_price"
            type="number"
            min="1"
            step="1"
            value={sessionPrice}
            onChange={(e) => setSessionPrice(e.target.value)}
          />

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium text-ink">
              Serviços oferecidos
            </span>
            <div className="flex flex-wrap gap-2">
              {catalogs.services.map((service) => (
                <ChipToggle
                  key={service.id}
                  label={service.name}
                  selected={selectedIds.includes(service.id)}
                  onClick={() => toggleSpecialty(service.id)}
                />
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium text-ink">
              Objetivos com os quais você trabalha
            </span>
            <div className="flex flex-wrap gap-2">
              {catalogs.objectives.map((objective) => (
                <ChipToggle
                  key={objective.id}
                  label={objective.name}
                  selected={objectiveIds.includes(objective.id)}
                  onClick={() =>
                    setObjectiveIds((prev) => toggleId(prev, objective.id))
                  }
                />
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium text-ink">
              Públicos atendidos
            </span>
            <div className="flex flex-wrap gap-2">
              {catalogs.audiences
                .filter((audience) => !audience.is_specific)
                .map((audience) => (
                  <ChipToggle
                    key={audience.id}
                    label={audience.name}
                    selected={audienceIds.includes(audience.id)}
                    onClick={() =>
                      setAudienceIds((prev) => toggleId(prev, audience.id))
                    }
                  />
                ))}
              <ChipToggle
                label="Atendimento a públicos específicos"
                selected={showSpecificAudiences}
                onClick={() => {
                  if (showSpecificAudiences) {
                    setAudienceIds((prev) =>
                      prev.filter((id) => !specificAudienceIds.includes(id)),
                    );
                    setAudienceOther("");
                    setAudienceExperience("");
                  }
                  setShowSpecificAudiences((prev) => !prev);
                }}
              />
            </div>

            {showSpecificAudiences && (
              <div className="mt-2 flex flex-wrap gap-2">
                {catalogs.audiences
                  .filter((audience) => audience.is_specific)
                  .map((audience) => (
                    <ChipToggle
                      key={audience.id}
                      label={audience.name}
                      selected={audienceIds.includes(audience.id)}
                      onClick={() =>
                        setAudienceIds((prev) => toggleId(prev, audience.id))
                      }
                    />
                  ))}
              </div>
            )}

            {otherAudienceSelected && (
              <div className="mt-2 flex flex-col gap-1.5">
                <label
                  htmlFor="audience_other"
                  className="text-sm font-medium text-ink"
                >
                  Informe qual público você possui experiência em atender.
                  <span className="text-brand"> *</span>
                </label>
                <textarea
                  id="audience_other"
                  name="audience_other"
                  rows={2}
                  required
                  value={audienceOther}
                  onChange={(e) => setAudienceOther(e.target.value)}
                  className="w-full rounded-xl border border-slate-300 bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
                />
              </div>
            )}

            {hasSpecificAudience && (
              <div className="mt-2 flex flex-col gap-1.5">
                <label
                  htmlFor="audience_experience"
                  className="text-sm font-medium text-ink"
                >
                  Conte brevemente sobre sua experiência ou formação para
                  atender esse público.
                </label>
                <textarea
                  id="audience_experience"
                  name="audience_experience"
                  rows={3}
                  value={audienceExperience}
                  onChange={(e) => setAudienceExperience(e.target.value)}
                  className="w-full rounded-xl border border-slate-300 bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
                />
              </div>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium text-ink">
              Locais de atendimento presencial
            </span>
            <div className="flex flex-wrap gap-2">
              {catalogs.locations.map((location) => (
                <ChipToggle
                  key={location.id}
                  label={location.name}
                  selected={locationIds.includes(location.id)}
                  onClick={() =>
                    setLocationIds((prev) => toggleId(prev, location.id))
                  }
                />
              ))}
            </div>
          </div>

          <NeighborhoodPicker
            selected={neighborhoods}
            onChange={setNeighborhoods}
            defaultCity={city}
            defaultState={state}
          />

          <div className="flex flex-col gap-1.5">
            <label htmlFor="bio" className="text-sm font-medium text-ink">
              Bio / apresentação
            </label>
            <textarea
              id="bio"
              name="bio"
              rows={4}
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              className="w-full rounded-xl border border-slate-300 bg-surface px-3.5 py-2.5 text-sm text-ink outline-none transition focus:border-brand focus:ring-2 focus:ring-brand/20"
            />
          </div>

          <Button type="submit" disabled={saving}>
            {saving ? "Salvando..." : "Salvar alterações"}
          </Button>
        </form>
      </Card>
    </div>
  );
}

export default ProfileView;
