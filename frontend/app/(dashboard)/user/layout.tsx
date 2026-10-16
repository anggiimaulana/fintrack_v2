import { Sidebar } from "@/components/layout/Sidebar";
import { MobileBottomNav } from "@/components/layout/MobileBottomNav";
import { DashboardShell } from "@/components/layout/DashboardShell";

export default async function UserDashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <DashboardShell
      sidebar={<Sidebar />}
      bottomNav={<MobileBottomNav />}
      role="user"
    >
      {children}
    </DashboardShell>
  );
}
