import { ProfessionalHeader } from "@/components/professional/ProfessionalHeader";

export default function ProfessionalLayout({
  children,
}: LayoutProps<"/app/professional">) {
  return (
    <div className="flex min-h-screen flex-1 flex-col">
      <ProfessionalHeader />
      <main className="flex-1">{children}</main>
    </div>
  );
}
