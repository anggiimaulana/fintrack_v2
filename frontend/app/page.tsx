import Link from "next/link";
import {
  MessageSquare, Mic, Camera, LayoutDashboard, Target, Shield,
  ArrowRight, Wallet, Star
} from "lucide-react";
import { Button } from "@/components/ui/button";

export default function LandingPage() {
  return (
    <div className="flex flex-col min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-50 w-full border-b-2 border-foreground bg-card">
        <div className="mx-auto max-w-7xl px-4 h-16 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Wallet className="w-6 h-6 text-primary" />
            <span className="text-xl font-black tracking-tight">
              Fin<span className="text-primary">Track</span>
            </span>
          </div>
          <nav className="hidden md:flex gap-6">
            <a href="#fitur" className="text-sm font-bold text-muted-foreground hover:text-foreground transition-colors uppercase tracking-wide">Fitur</a>
            <a href="#cara-kerja" className="text-sm font-bold text-muted-foreground hover:text-foreground transition-colors uppercase tracking-wide">Cara Kerja</a>
            <a href="#harga" className="text-sm font-bold text-muted-foreground hover:text-foreground transition-colors uppercase tracking-wide">Harga</a>
          </nav>
          <div className="hidden md:flex items-center gap-3">
            <Button variant="outline" asChild>
              <Link href="/login">Masuk</Link>
            </Button>
            <Button asChild>
              <Link href="/register">Mulai Gratis</Link>
            </Button>
          </div>
          <button className="md:hidden p-2 text-foreground cursor-pointer">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        </div>
      </header>

      <main className="flex-1">
        <section className="relative bg-surface-dark overflow-hidden pt-20 pb-32 border-b-2 border-foreground">
          <div className="mx-auto max-w-7xl px-4 relative z-10 flex flex-col lg:flex-row items-center gap-12">
            <div className="flex-1 text-center lg:text-left space-y-8">
              <div className="inline-flex items-center border-2 border-primary bg-primary/20 px-3 py-1 text-sm font-bold text-white">
                <span className="flex h-2 w-2 bg-primary mr-2 animate-pulse" />
                Multi AI Provider
              </div>
              <h1 className="text-4xl md:text-5xl lg:text-6xl font-black leading-tight text-white">
                Kelola Keuangan Cerdas dengan <span className="text-primary">Perintah Natural</span>
              </h1>
              <p className="text-lg text-neutral-400 max-w-2xl mx-auto lg:mx-0 font-medium">
                Catat transaksi via teks, suara, atau foto. AI kami memahami bahasa Indonesia dan langsung menyimpan ke database Anda.
              </p>
              <div className="flex flex-col sm:flex-row items-center justify-center lg:justify-start gap-4">
                <Button size="lg" className="w-full sm:w-auto bg-primary text-white" asChild>
                  <Link href="/register">Mulai Gratis <ArrowRight className="ml-2 h-4 w-4" /></Link>
                </Button>
                <Button size="lg" variant="outline" className="w-full sm:w-auto bg-transparent border-white text-white hover:bg-white/10">
                  Lihat Demo
                </Button>
              </div>
            </div>
            <div className="flex-1 w-full max-w-xl lg:max-w-none">
              <div className="border-2 border-foreground bg-surface-dark-2 shadow-[6px_6px_0px_0px] shadow-primary/50 overflow-hidden">
                <div className="h-8 border-b-2 border-foreground bg-surface-dark-3 flex items-center px-4 gap-2">
                  <div className="w-3 h-3 border border-foreground bg-red-500 rounded-full" />
                  <div className="w-3 h-3 border border-foreground bg-amber-500 rounded-full" />
                  <div className="w-3 h-3 border border-foreground bg-green-500 rounded-full" />
                </div>
                <div className="flex-1 p-6 bg-surface-dark-2 flex flex-col gap-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div className="border-2 border-foreground bg-surface-dark-3 p-4">
                      <div className="text-neutral-400 text-xs font-bold uppercase mb-1">Total Pemasukan</div>
                      <div className="text-green-400 font-bold font-mono text-lg">Rp 12.500.000</div>
                    </div>
                    <div className="border-2 border-foreground bg-surface-dark-3 p-4">
                      <div className="text-neutral-400 text-xs font-bold uppercase mb-1">Total Pengeluaran</div>
                      <div className="text-red-400 font-bold font-mono text-lg">Rp 4.200.000</div>
                    </div>
                  </div>
                  <div className="border-2 border-foreground bg-surface-dark-3 p-4 flex flex-col">
                    <div className="flex items-center gap-2 mb-4">
                      <div className="w-8 h-8 bg-primary flex items-center justify-center">
                        <MessageSquare className="w-4 h-4 text-white" />
                      </div>
                      <div className="text-sm font-bold text-white">Asisten Keuangan</div>
                    </div>
                    <div className="bg-primary text-white text-sm p-3 self-end max-w-[80%] mb-4 font-medium">
                      Tolong catat beli kopi 25rb tadi siang
                    </div>
                    <div className="border-2 border-foreground bg-surface-dark-2 text-white text-sm p-3 self-start max-w-[80%] font-medium">
                      Siapp! Pengeluaran Rp 25.000 untuk &quot;Kopi&quot; telah dicatat.
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section id="fitur" className="py-24 bg-card border-b-2 border-foreground">
          <div className="mx-auto max-w-7xl px-4">
            <div className="text-center max-w-2xl mx-auto mb-16">
              <h2 className="text-3xl md:text-4xl font-black uppercase tracking-wide mb-4">Fitur Unggulan</h2>
              <p className="text-muted-foreground font-medium">FinTrack menghadirkan cara baru mencatat keuangan yang lebih mudah dan cerdas.</p>
            </div>
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {[
                { icon: MessageSquare, title: "AI Chat CRUD", desc: "Catat transaksi dengan mengetik biasa. AI otomatis mengekstrak kategori dan jumlah." },
                { icon: Mic, title: "Input Suara", desc: "Sedang sibuk? Ucapkan transaksi Anda, biar AI yang mencatat ke sistem." },
                { icon: Camera, title: "Foto Struk", desc: "Foto bon belanja Anda, AI akan mengekstrak semua item otomatis." },
                { icon: LayoutDashboard, title: "Dashboard Interaktif", desc: "Pantau grafik dan statistik real-time untuk keputusan keuangan lebih baik." },
                { icon: Target, title: "Analisis SPK", desc: "Sistem cerdas mengklasifikasi kondisi keuangan Anda dari Sangat Baik hingga Kritis." },
                { icon: Shield, title: "Privasi Total", desc: "Data tersimpan aman. AI key milik Anda sendiri, kami tidak membaca data Anda." },
              ].map((feature, i) => (
                <div key={i} className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6 hover:shadow-[6px_6px_0px_0px] hover:shadow-foreground transition-all duration-100">
                  <div className="w-12 h-12 border-2 border-foreground bg-primary/10 flex items-center justify-center mb-6">
                    <feature.icon className="w-6 h-6 text-primary" />
                  </div>
                  <h3 className="text-lg font-black uppercase tracking-wide mb-2">{feature.title}</h3>
                  <p className="text-muted-foreground text-sm font-medium leading-relaxed">{feature.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section id="cara-kerja" className="py-24 bg-muted border-b-2 border-foreground">
          <div className="mx-auto max-w-7xl px-4">
            <div className="text-center max-w-2xl mx-auto mb-16">
              <h2 className="text-3xl md:text-4xl font-black uppercase tracking-wide mb-4">Cara Kerja FinTrack</h2>
              <p className="text-muted-foreground font-medium">Mulai kelola keuangan Anda dalam 3 langkah mudah.</p>
            </div>
            <div className="grid md:grid-cols-3 gap-6">
              {[
                { step: "1", title: "Tambahkan API Key", desc: "Gunakan API key dari berbagai provider AI untuk privasi penuh dan akses tak terbatas." },
                { step: "2", title: "Catat Transaksi", desc: "Teks, suara, atau foto. Beritahu AI transaksi Anda senatural mungkin." },
                { step: "3", title: "Analisis Keuangan", desc: "Tanyakan apapun soal keuangan Anda, AI akan menjawab berdasarkan data Anda." },
              ].map((item, i) => (
                <div key={i} className="border-2 border-foreground bg-card shadow-[5px_5px_0px_0px] shadow-foreground p-6 text-center">
                  <div className="w-12 h-12 border-2 border-foreground bg-primary text-primary-foreground flex items-center justify-center text-xl font-black mx-auto mb-6 shadow-[3px_3px_0px_0px] shadow-foreground">
                    {item.step}
                  </div>
                  <h3 className="text-base font-black uppercase tracking-wide mb-3">{item.title}</h3>
                  <p className="text-sm text-muted-foreground font-medium">{item.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="py-24 bg-card border-b-2 border-foreground">
          <div className="mx-auto max-w-7xl px-4 flex flex-col lg:flex-row items-center gap-12">
            <div className="flex-1 space-y-6">
              <h2 className="text-3xl md:text-4xl font-black uppercase tracking-wide">Ketahui Kondisi Keuangan Anda</h2>
              <p className="text-muted-foreground text-lg font-medium">
                Sistem Pendukung Keputusan (SPK) kami menganalisis pola pengeluaran, tabungan, dan kebiasaan Anda untuk memberikan skor dan status keuangan.
              </p>
              <ul className="space-y-4">
                {[
                  { status: "Sangat Baik", color: "bg-green-500" },
                  { status: "Baik", color: "bg-primary" },
                  { status: "Cukup", color: "bg-amber-500" },
                  { status: "Boros", color: "bg-orange-500" },
                  { status: "Kritis", color: "bg-red-600" },
                ].map((item, i) => (
                  <li key={i} className="flex items-center gap-3">
                    <div className={`w-3 h-3 border border-foreground ${item.color}`} />
                    <span className="font-bold text-foreground">{item.status}</span>
                  </li>
                ))}
              </ul>
            </div>
            <div className="flex-1 w-full border-2 border-foreground bg-muted p-8 shadow-[5px_5px_0px_0px] shadow-foreground">
              <div className="border-2 border-foreground bg-card shadow-[3px_3px_0px_0px] shadow-foreground p-6 flex items-center gap-6">
                <div className="relative w-24 h-24 flex items-center justify-center border-2 border-foreground bg-green-50">
                  <svg className="absolute inset-0 w-full h-full" viewBox="0 0 100 100">
                    <circle cx="50" cy="50" r="46" fill="transparent" stroke="#e5e7eb" strokeWidth="8" />
                    <circle cx="50" cy="50" r="46" fill="transparent" stroke="#16A34A" strokeWidth="8" strokeDasharray="289" strokeDashoffset="40" />
                  </svg>
                  <span className="text-2xl font-black text-green-600">85</span>
                </div>
                <div>
                  <h4 className="text-xl font-black uppercase tracking-wide text-foreground mb-1">Sangat Baik</h4>
                  <p className="text-sm text-muted-foreground font-medium">Kondisi keuangan Anda stabil. Rasio tabungan Anda 30% dari pendapatan.</p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="py-24 bg-muted border-b-2 border-foreground">
          <div className="mx-auto max-w-7xl px-4">
            <h2 className="text-3xl md:text-4xl font-black uppercase tracking-wide text-center mb-16">Apa Kata Mereka?</h2>
            <div className="grid md:grid-cols-3 gap-6">
              {[
                { name: "Budi Santoso", role: "Freelancer", text: "Mencatat pengeluaran kini semudah chat dengan teman. Sangat terbantu!" },
                { name: "Siti Aminah", role: "Ibu Rumah Tangga", text: "Fitur foto struk sangat ajaib! Semua belanjaan langsung tercatat otomatis." },
                { name: "Andi Wijaya", role: "Karyawan Swasta", text: "Analisis SPK bikin saya sadar kalau bulan ini saya terlalu boros di kopi." },
              ].map((testi, i) => (
                <div key={i} className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6">
                  <div className="flex gap-1 mb-4">
                    {[1,2,3,4,5].map(star => (
                      <Star key={star} className="w-4 h-4 fill-amber-500 text-amber-500" />
                    ))}
                  </div>
                  <p className="text-muted-foreground font-medium mb-6 italic">&ldquo;{testi.text}&rdquo;</p>
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 border-2 border-foreground bg-primary flex items-center justify-center text-primary-foreground font-black">
                      {testi.name.charAt(0)}
                    </div>
                    <div>
                      <h4 className="font-black text-sm uppercase tracking-wide">{testi.name}</h4>
                      <p className="text-xs text-muted-foreground font-bold">{testi.role}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="py-24 border-b-2 border-foreground bg-primary text-primary-foreground text-center">
          <div className="mx-auto max-w-7xl px-4">
            <h2 className="text-3xl md:text-4xl lg:text-5xl font-black uppercase tracking-wide mb-6">Mulai Kelola Keuangan Anda Sekarang</h2>
            <p className="text-lg font-medium text-primary-foreground/80 mb-10 max-w-2xl mx-auto">
              Bergabunglah dengan ribuan pengguna lain yang telah merasakan kemudahan mencatat keuangan dengan FinTrack.
            </p>
            <Button size="lg" variant="outline" className="bg-white text-primary hover:bg-neutral-100 border-foreground font-black" asChild>
              <Link href="/register">Daftar Sekarang <ArrowRight className="ml-2 w-4 h-4" /></Link>
            </Button>
          </div>
        </section>
      </main>

      <footer className="border-t-2 border-foreground bg-surface-dark text-neutral-400 py-12">
        <div className="mx-auto max-w-7xl px-4 flex flex-col md:flex-row justify-between items-center gap-6">
          <div className="flex flex-col gap-2 items-center md:items-start">
            <span className="text-xl font-black tracking-tight text-white flex items-center gap-2">
              <Wallet className="w-6 h-6 text-primary" />
              Fin<span className="text-primary">Track</span>
            </span>
            <span className="text-sm font-medium">Catat. Tanya. Kelola. Semuanya dalam satu perintah.</span>
          </div>
          <div className="flex gap-6 text-sm font-bold">
            <a href="#fitur" className="hover:text-white transition-colors uppercase tracking-wide">Fitur</a>
            <a href="#tentang" className="hover:text-white transition-colors uppercase tracking-wide">Tentang</a>
            <a href="#kontak" className="hover:text-white transition-colors uppercase tracking-wide">Kontak</a>
            <a href="#privasi" className="hover:text-white transition-colors uppercase tracking-wide">Kebijakan Privasi</a>
          </div>
        </div>
        <div className="mx-auto max-w-7xl px-4 mt-8 pt-8 border-t border-neutral-800 text-sm font-bold text-center md:text-left uppercase tracking-wide">
          &copy; 2026 FinTrack. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
