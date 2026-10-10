'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { login as authLogin } from '@/lib/auth-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Eye, EyeOff, Loader2, LogIn, Wallet } from 'lucide-react'

export default function LoginPage() {
  const router = useRouter()

  
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError(null)

    try {
      const user = await authLogin(email, password)

      if (!user.is_active) {
        throw new Error('Akun Anda telah dinonaktifkan. Hubungi admin.')
      }

      const redirectPath = user.role === 'admin' ? '/admin' : '/user'
      router.push(redirectPath)
      router.refresh()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : ''
      if (message.includes('429') || message.toLowerCase().includes('terlalu banyak')) {
        setError('Terlalu banyak percobaan masuk. Silakan coba lagi nanti.')
      } else if (message.includes('401') || message.toLowerCase().includes('salah')) {
        setError('Email atau password salah.')
      } else {
        setError(message || 'Email atau password salah.')
      }
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[#F3F4F6] px-4 py-12">
      <div className="w-full max-w-md">
        <div className="flex justify-center mb-8">
          <div className="flex items-center gap-2 text-3xl font-black tracking-tight text-foreground">
            <Wallet className="w-8 h-8 text-primary" />
            Fin<span className="text-primary">Track</span>
          </div>
        </div>
        <Card>
          <CardHeader className="text-center">
            <CardTitle className="text-xl font-black uppercase tracking-wide">
              Masuk ke Akun
            </CardTitle>
            <p className="text-sm text-muted-foreground">
              Masukkan email dan password untuk melanjutkan
            </p>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleLogin} className="space-y-5">
              {error && (
                <div className="border-2 border-destructive bg-destructive/10 p-3 text-sm font-medium text-destructive">
                  {error}
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="nama@perusahaan.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  disabled={isLoading}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Masukkan password"
                    required
                    disabled={isLoading}
                    className="pr-10"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
                  >
                    {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                  </button>
                </div>
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={isLoading}
              >
                {isLoading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Memproses...
                  </>
                ) : (
                  <>
                    <LogIn className="mr-2 h-4 w-4" />
                    Masuk
                  </>
                )}
              </Button>
              <div className="text-center text-sm text-muted-foreground">
                Belum punya akun? <Link href="/register" className="font-bold text-primary hover:underline">Daftar Sekarang</Link>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
