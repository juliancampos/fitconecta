import { Header } from "@/components/Header";

export default function ClientLayout({ children }: LayoutProps<"/app/client">) {
  return (
    <div className="flex min-h-screen flex-1 flex-col">
      <Header />
      <main className="flex-1">{children}</main>
    </div>
  );
}
