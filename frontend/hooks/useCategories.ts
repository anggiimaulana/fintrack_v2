import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export function useIncomeCategories() {
  return useQuery({
    queryKey: queryKeys.categories.income,
    queryFn: async () => {
      const res = await api.get("/categories/income");
      return res.data?.data || [];
    },
    staleTime: staleTimeConfig.categories,
  });
}

export function useExpenseCategories() {
  return useQuery({
    queryKey: queryKeys.categories.expense,
    queryFn: async () => {
      const res = await api.get("/categories/expense");
      return res.data?.data || [];
    },
    staleTime: staleTimeConfig.categories,
  });
}

export function useSubcategories(categoryId: string) {
  return useQuery({
    queryKey: queryKeys.categories.subcategories(categoryId),
    queryFn: async () => {
      const res = await api.get(`/categories/expense/${categoryId}/subcategories`);
      return res.data?.data || [];
    },
    staleTime: staleTimeConfig.categories,
    enabled: !!categoryId,
  });
}
