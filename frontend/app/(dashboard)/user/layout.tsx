import { Sidebar } from "@/components/layout/Sidebar";
import { Header } from "@/components/layout/Header";
import { MobileBottomNav } from "@/components/layout/MobileBottomNav";

export default async function UserDashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-screen w-full bg-background overflow-hidden text-foreground font-sans">
      <Sidebar />
      <div className="flex flex-col flex-1 min-w-0 h-full overflow-hidden">
        <Header />
        {/* Bottom padding clears the fixed mobile nav (h-16) plus the iOS home
            indicator, so the last row of content is never trapped underneath. */}
        <main className="flex-1 overflow-y-auto p-4 lg:p-6 pb-[calc(5rem+env(safe-area-inset-bottom))] md:pb-6">
          {children}
        </main>
      </div>
      <MobileBottomNav />
    </div>
  );
}
