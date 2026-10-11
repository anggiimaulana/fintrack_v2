import { SidebarProvider } from '@/components/ui/sidebar'
import { SuperAdminSidebar } from '@/components/super-admin/SuperAdminSidebar'

export default async function SuperAdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <SidebarProvider>
      <div className="flex min-h-screen w-full bg-background">
        <SuperAdminSidebar />
        <main className="flex-1 flex flex-col min-w-0">
          <div className="flex-1 p-6 md:p-8 overflow-y-auto">
            {children}
          </div>
        </main>
      </div>
    </SidebarProvider>
  )
}
