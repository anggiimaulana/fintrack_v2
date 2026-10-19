# Spesifikasi Teknis: Kontrol Form, Validasi Reaktif, Format Rupiah & Dialog Konfirmasi

Dokumen ini mendefinisikan standar input numerik mata uang, aturan batas minimum angka, sistem validasi reaktif tombol aksi, dan dialog modal konfirmasi untuk operasi kritis pada portal `admin` dan `super_admin`.

---

## 1. Komponen `RupiahInput` Terstandarisasi

Komponen `RupiahInput` (`@/components/shared/RupiahInput.tsx`) bertindak sebagai standar tunggal untuk seluruh input nilai moneter (harga paket, nominal penyesuaian, saldo, limit kuota nominal):

### 1.1. Perilaku Pengetikan & Format Otomatis
1. **Pemisah Ribuan Otomatis**:
   - Mengetik `50000` langsung terformat di antarmuka menjadi `50.000`.
   - Pemisah ribuan menggunakan karakter titik (`.`) sesuai kaidah Bahasa Indonesia.
   - Karakter non-digit otomatis diabaikan (*sanitized regex* `/[^0-9]/g`).
2. **Visual Prefix Tetap**:
   - Prefix visual `Rp` disematkan di sisi kiri input dengan posisi absolut.
   - Nilai teks input memiliki padding kiri `pl-9` sehingga digit angka tidak pernah menimpa label `Rp`.
3. **Nilai Kembalian Mentah (*Raw Value*)**:
   - Properti `onChange` selalu mengembalikan tipe data `number` murni (atau `undefined` jika kosong), **bukan string terformat**.
   - Komponen pengonsumsi tidak perlu melakukan parsing string manual (`parseInt` atau `replace`).

```typescript
export interface RupiahInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value?: number;
  onChange?: (value: number | undefined) => void;
  min?: number; // Default 1
  allowZero?: boolean; // Default false
  errorMessage?: string;
}
```

---

## 2. Batas Minimum Angka & Penegakan Validasi

### 2.1. Penegakan Nilai Minimum $\ge 1$
Sesuai aturan sistem FinTrack v2, seluruh input numerik pada form admin/super admin (seperti alokasi kuota token, harga paket bulanan/tahunan, dan batas nominal) wajib bernilai **minimal 1**:
- Input nilai `0`, nilai negatif, atau desimal di bawah 1 langsung dianggap **tidak valid**.
- Pada event `onBlur` atau saat form divalidasi, jika nilai `< 1`, helper pesan kesalahan merah ditampilkan di bawah input:
  *"Nilai minimal adalah 1"*.

### 2.2. Validasi Reaktif & Guard Tombol Aksi Form
Semua tombol simpan, submit, atau langkah berikutnya (Next) wajib dilindungi guard reaktif:

```tsx
<Button
  type="submit"
  disabled={!isFormValid || isSubmitting}
  className="disabled:opacity-50 disabled:cursor-not-allowed"
>
  {isSubmitting ? (
    <>
      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
      Menyimpan...
    </>
  ) : (
    "Simpan Perubahan"
  )}
</Button>
```

### 2.3. Kriteria Status `isFormValid`
Tombol aksi hanya akan aktif (`disabled === false`) jika seluruh kondisi berikut terpenuhi secara simultan:
1. **Kelengkapan Field Wajib**: Semua input yang memiliki atribut `required` tidak boleh kosong atau hanya berisi spasi (`field.trim().length > 0`).
2. **Integritas Numerik**: Semua field input numerik memiliki nilai valid bertipe integer dengan nilai $\ge 1$.
3. **Validitas Format Khusus**: Email sesuai regex RFC 5322, URL diawali `https://`, dan skema warna hex valid.
4. **Anti-Double-Click**: Saat request mutasi dikirim (`isSubmitting === true` atau `mutation.isPending === true`), tombol terkunci seketika untuk mencegah duplikasi eksekusi request pada jaringan lambat.

---

## 3. Komponen Dialog Konfirmasi Aksi Destruktif (`ConfirmActionModal`)

Untuk mencegah kesalahan operasional fatal (*human error*), seluruh aksi destruktif (menghapus, menangguhkan/suspend, atau menonaktifkan data) wajib melewati dialog modal konfirmasi terstandarisasi (`@/components/shared/ConfirmActionModal.tsx`).

### 3.1. Variasi Aksi & Visual Semantik

| Tipe Aksi | Warna Aksen | Ikon SVG Resmi | Tombol Aksi | Skenario Penggunaan |
|---|---|---|---|---|
| `delete` | Destructive (Merah) | `Trash2` | Merah Solid | Hapus paket plan, hapus kategori permanen, hapus API key. |
| `suspend` | Warning (Oranye / Amber) | `ShieldAlert` | Oranye Solid | Suspend akun pengguna karena fraud, isolasi rute gateway. |
| `deactivate` | Secondary (Netral Abu) | `PowerOff` | Dark / Slate | Nonaktifkan provider pembayaran, arsipkan kategori. |

### 3.2. Struktur Antarmuka Komponen
```typescript
export interface ConfirmActionModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  actionType: "delete" | "suspend" | "deactivate";
  title: string;
  description: string;
  entityName?: string; // e.g. "John Doe" atau "Paket Pro Tahunan"
  confirmLabel?: string;
  cancelLabel?: string;
  isPending?: boolean;
  onConfirm: () => void | Promise<void>;
}
```

### 3.3. Tampilan Modal Konfirmasi
```
+---------------------------------------------------------------------------------------------------------+
| [Ikon SVG ShieldAlert]   Suspend Akun Pengguna                                                          |
|                                                                                                         |
| Apakah Anda yakin ingin menangguhkan (suspend) akun untuk "John Doe (john@example.com)"?               |
|                                                                                                         |
| Tindakan ini akan mencabut seluruh akses login, memblokir penggunaan API gateway,                       |
| dan membekukan langganan aktif pengguna sampai peninjauan selesai.                                      |
|                                                                                                         |
|                                             [Batal (Outline)]   [Ya, Suspend Akun (Amber Solid)]        |
+---------------------------------------------------------------------------------------------------------+
```

---

## 4. Standar Responsivitas Form & Anti-Overflow

1. **Tata Letak Grid Responsif**:
   - Field form disusun dengan sistem grid adaptif: `grid grid-cols-1 md:grid-cols-2 gap-4`.
   - Field teks panjang atau editor JSON menggunakan bentang penuh `md:col-span-2`.
2. **Kontrol Batas Tinggi & Scrollbar Dialog**:
   - Seluruh `<DialogContent>` menerapkan batasan:
     `max-h-[85vh] overflow-y-auto px-6 py-6 sm:max-w-[600px]`.
   - Mencegah elemen form, dropdown selector, atau tombol aksi keluar dari batas viewport layar pada monitor resolusi rendah atau laptop kecil.
3. **Pemberitahuan Validasi Aksesibel**:
   - Pesan error menggunakan elemen `<p role="alert" className="text-xs font-semibold text-destructive mt-1">` dengan atribut `aria-live="polite"`.
