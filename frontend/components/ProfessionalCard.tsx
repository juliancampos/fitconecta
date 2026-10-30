import Link from "next/link";
import Image from "next/image";
import type { ProfessionalProfile } from "@/lib/types";

function initials(name: string) {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join("");
}

export function ProfessionalCard({
  professional,
}: {
  professional: ProfessionalProfile;
}) {
  const specialtyNames = (professional.specialties ?? []).map((s) => s.name);
  const neighborhoodNames = (professional.neighborhoods ?? []).map(
    (n) => n.name,
  );

  return (
    <article className="flex flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
      <div className="flex items-center gap-4 border-b border-slate-100 p-5">
        <div className="relative flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-full bg-brand-soft text-lg font-bold text-brand">
          {professional.avatar_url ? (
            <Image
              src={professional.avatar_url}
              alt={professional.display_name}
              width={56}
              height={56}
              className="h-full w-full object-cover"
            />
          ) : (
            initials(professional.display_name)
          )}
        </div>
        <div className="min-w-0">
          <h3 className="truncate font-semibold text-ink">
            {professional.display_name}
          </h3>
          <p className="truncate text-sm text-muted">
            {professional.city}, {professional.state}
          </p>
        </div>
        {professional.reviews > 0 && (
          <div className="ml-auto flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-sm font-semibold text-amber-600">
            <span aria-hidden>★</span>
            {professional.rating.toFixed(1)}
          </div>
        )}
      </div>

      <div className="flex flex-1 flex-col gap-3 p-5">
        <div className="flex flex-wrap gap-1.5">
          {specialtyNames.map((specialty) => (
            <span
              key={specialty}
              className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-600"
            >
              {specialty}
            </span>
          ))}
        </div>

        {neighborhoodNames.length > 0 && (
          <p className="text-xs text-muted">
            <span aria-hidden className="mr-1">
              📍
            </span>
            Atende: {neighborhoodNames.slice(0, 4).join(", ")}
            {neighborhoodNames.length > 4
              ? ` +${neighborhoodNames.length - 4}`
              : ""}
          </p>
        )}

        {professional.bio && (
          <p className="line-clamp-2 text-sm text-muted">{professional.bio}</p>
        )}

        <div className="mt-auto flex items-end justify-between pt-2">
          <div>
            <span className="text-lg font-bold text-ink">
              R$ {(professional.session_price_cents / 100).toFixed(0)}
            </span>
            <span className="text-xs text-muted">/sessão</span>
          </div>
          <Link
            href={`/app/client/profissional/${professional.id}`}
            className="inline-flex items-center justify-center gap-2 rounded-xl bg-brand px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-brand-dark"
          >
            Ver perfil
          </Link>
        </div>

        {professional.reviews > 0 && (
          <p className="text-xs text-muted">
            {professional.reviews} avaliações
          </p>
        )}
      </div>
    </article>
  );
}

export default ProfessionalCard;
