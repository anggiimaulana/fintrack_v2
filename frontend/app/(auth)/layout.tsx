import { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'Authentication | FinTrack',
  description: 'Masuk ke akun FinTrack Anda',
}

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="min-h-screen flex flex-col">
      <main className="flex-1 flex items-center justify-center">
        {children}
      </main>
      <footer className="py-6 text-center text-xs font-bold text-muted-foreground uppercase tracking-wide">
        &copy; {new Date().getFullYear()} FinTrack. All rights reserved.
      </footer>
    </div>
  )
}
