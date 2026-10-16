import { DashboardShell } from '@/components/layout/DashboardShell'
import { AdminSidebar } from '@/components/admin/AdminSidebar'

export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <DashboardShell sidebar={<AdminSidebar />} requiredRole="admin">
      {children}
    </DashboardShell>
  )
}
