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
  const isNutrition = professional.branch === "nutrition";
  const councilName = isNutrition ? "CRN" : "CREF";
  const councilNumber = isNutrition
    ? professional.crn_number
    : professional.cref_number;
  const councilVerified = Boolean(
    isNutrition ? professional.crn_verified_at : professional.cref_verified_at,
  );
  const councilSource = isNutrition
    ? "na Consulta Nacional do CFN"
    : "no portal do CREF6";

  return (
    <article className="flex flex-col overflow-hidden rounded-2xl border border-slate-200 bg-surface shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
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
          {councilNumber && (
            <span
              title={
                councilVerified
                  ? `${councilName} validado ${councilSource}`
                  : `${councilName} ainda não confirmado ${councilSource}`
              }
              className={`mt-1 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold ${
                councilVerified
                  ? "bg-emerald-50 text-emerald-600"
                  : "bg-amber-50 text-amber-600"
              }`}
            >
              {councilVerified ? (
                <svg
                  aria-hidden="true"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className="h-3.5 w-3.5"
                >
                  <path
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.857-9.809a.75.75 0 0 0-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 1 0-1.06 1.061l2.5 2.5a.75.75 0 0 0 1.137-.089l4-5.5Z"
                  />
                </svg>
              ) : (
                <svg
                  aria-hidden="true"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  className="h-3.5 w-3.5"
                >
                  <path
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm.75-13a.75.75 0 0 0-1.5 0v5c0 .414.336.75.75.75h4a.75.75 0 0 0 0-1.5h-3.25V5Z"
                  />
                </svg>
              )}
              {councilVerified
                ? `${councilName} verificado`
                : `${councilName} em análise`}
            </span>
          )}
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-3 p-5">
        {/* Linha propria para a tag do ramo nao tampar o nome. */}
        <div className="flex items-center justify-between">
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold ${
              isNutrition
                ? "bg-emerald-50 text-emerald-600"
                : "bg-brand-soft text-brand"
            }`}
          >
            {isNutrition ? (
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-3 w-3"
              >
                <path d="M12 8c0-2 1-3.5 3-4M12 8c-4.5 0-7 3-7 8 0 3 2 5 4 5 1.5 0 2-1 3-1s1.5 1 3 1c2 0 4-2 4-5 0-5-2.5-8-7-8z" />
              </svg>
            ) : (
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                className="h-3 w-3"
              >
                <path d="M6.5 6.5v11M17.5 6.5v11M3.5 9.5v5M20.5 9.5v5M6.5 12h11" />
              </svg>
            )}
            {isNutrition ? "Nutrição" : "Educação Física"}
          </span>
          {professional.reviews > 0 && (
            <div className="flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-sm font-semibold text-amber-600">
              <span aria-hidden>★</span>
              {professional.rating.toFixed(1)}
            </div>
          )}
        </div>

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
