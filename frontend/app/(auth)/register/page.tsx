'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { register as authRegister } from '@/lib/auth-client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Eye, EyeOff, Loader2, UserPlus, Wallet, CheckCircle } from 'lucide-react'

export default function RegisterPage() {
  const router = useRouter()

  
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isSuccess, setIsSuccess] = useState(false)

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError(null)

    if (password !== confirmPassword) {
      setError('Password dan konfirmasi password tidak cocok.')
      setIsLoading(false)
      return
    }

    if (password.length < 8) {
      setError('Password minimal 8 karakter.')
      setIsLoading(false)
      return
    }

    try {
      await authRegister(email, password, '')
      setIsSuccess(true)
      setTimeout(() => {
        router.push('/login')
      }, 3000)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal mendaftar. Silakan coba lagi.')
      setIsLoading(false)
    }
  }

  if (isSuccess) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted px-4 py-12">
        <div className="w-full max-w-md">
          <div className="flex justify-center mb-8">
            <div className="flex items-center gap-2 text-3xl font-black tracking-tight text-foreground">
              <Wallet className="w-8 h-8 text-primary" />
              Fin<span className="text-primary">Track</span>
            </div>
          </div>
          <Card>
            <CardContent className="pt-8 flex flex-col items-center text-center space-y-4">
              <div className="w-16 h-16 border-2 rounded-md border-foreground bg-green-100 flex items-center justify-center shadow-[3px_3px_0px_0px] shadow-foreground">
                <CheckCircle className="w-8 h-8 text-green-600" />
              </div>
              <CardTitle className="text-xl font-black uppercase tracking-wide">Pendaftaran Berhasil</CardTitle>
              <p className="text-sm text-muted-foreground">
                Akun Anda telah berhasil dibuat. Anda akan dialihkan ke halaman login dalam beberapa detik.
              </p>
              <Button className="w-full" onClick={() => router.push('/login')}>
                Ke Halaman Login Sekarang
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted px-4 py-12">
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
              Daftar Akun Baru
            </CardTitle>
            <p className="text-sm text-muted-foreground">
              Lengkapi data di bawah untuk membuat akun
            </p>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleRegister} className="space-y-5">
              {error && (
                <div className="border-2 rounded-md border-destructive bg-destructive/10 p-3 text-sm font-medium text-destructive">
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
                    placeholder="Buat password (min 6 karakter)"
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
              <div className="space-y-2">
                <Label htmlFor="confirmPassword">Konfirmasi Password</Label>
                <Input
                  id="confirmPassword"
                  type={showPassword ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Ulangi password yang sama"
                  required
                  disabled={isLoading}
                />
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
                    <UserPlus className="mr-2 h-4 w-4" />
                    Daftar
                  </>
                )}
              </Button>
              <div className="text-center text-sm text-muted-foreground">
                Sudah punya akun? <Link href="/login" className="font-bold text-primary hover:underline">Masuk di sini</Link>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
