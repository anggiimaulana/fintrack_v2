import { DashboardShell } from '@/components/layout/DashboardShell'
import { SuperAdminSidebar } from '@/components/super-admin/SuperAdminSidebar'
import { GatewayAlertListener } from '@/components/super-admin/gateway/GatewayAlertListener'

export default async function SuperAdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <DashboardShell
      sidebar={<SuperAdminSidebar />}
      banner={<GatewayAlertListener />}
      requiredRole="super_admin"
    >
      {children}
    </DashboardShell>
  )
}
