'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { IconPicker } from './IconPicker'
import { ColorPicker } from './ColorPicker'
import { Loader2 } from 'lucide-react'

const categorySchema = z.object({
  name: z.string().min(2, 'Nama minimal 2 karakter'),
  icon: z.string().min(1, 'Pilih ikon'),
  color: z.string().min(1, 'Pilih warna'),
  is_active: z.boolean(),
})

type CategoryFormValues = z.infer<typeof categorySchema>

interface CategoryFormProps {
  initialValues?: Partial<CategoryFormValues> & { id?: string }
  onSubmit: (values: CategoryFormValues) => Promise<void>
  onCancel: () => void
  isLoading?: boolean
}

export function CategoryForm({ initialValues, onSubmit, onCancel, isLoading }: CategoryFormProps) {
  const form = useForm<CategoryFormValues>({
    resolver: zodResolver(categorySchema),
    mode: 'onChange',
    defaultValues: {
      name: initialValues?.name || '',
      icon: initialValues?.icon || 'Wallet',
      color: initialValues?.color || '#3b5bdb',
      is_active: initialValues?.is_active ?? true,
    },
  })

  const handleSubmit = async (values: CategoryFormValues) => {
    await onSubmit(values)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Nama Kategori</FormLabel>
              <FormControl>
                <Input placeholder="Contoh: Gaji, Makanan" {...field} disabled={isLoading} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="icon"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Pilih Ikon</FormLabel>
              <FormControl>
                <IconPicker value={field.value} onChange={field.onChange} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="color"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Pilih Warna</FormLabel>
              <FormControl>
                <ColorPicker value={field.value} onChange={field.onChange} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="is_active"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Status</FormLabel>
              <Select
                disabled={isLoading}
                onValueChange={(value: string) => field.onChange(value === 'active')}
                defaultValue={field.value ? 'active' : 'inactive'}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Pilih status" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="active">Aktif</SelectItem>
                  <SelectItem value="inactive">Nonaktif</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-neutral-500 mt-1">
                Kategori nonaktif tidak akan muncul di dropdown user.
              </p>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="flex justify-end gap-3 pt-4">
          <Button type="button" variant="ghost" onClick={onCancel} disabled={isLoading}>
            Batal
          </Button>
          <Button
            type="submit"
            disabled={!form.formState.isValid || isLoading}
            className="bg-primary hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed font-bold border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
          >
            {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Simpan Kategori
          </Button>
        </div>
      </form>
    </Form>
  )
}
