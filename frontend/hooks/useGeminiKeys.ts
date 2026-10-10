import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export function useGeminiKeys() {
  return useQuery({
    queryKey: queryKeys.geminiKeys.all,
    queryFn: async () => {
      const res = await api.get("/gemini-keys");
      return res.data?.data || [];
    },
    staleTime: staleTimeConfig.geminiKeys,
  });
}

export function useCreateGeminiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: { alias: string; key: string; provider?: string }) => {
      const res = await api.post("/gemini-keys", data);
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.geminiKeys.all });
    },
  });
}

export function useDeleteGeminiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/gemini-keys/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.geminiKeys.all });
    },
  });
}

export function useTestGeminiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await api.post(`/gemini-keys/${id}/test`);
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.geminiKeys.all });
    },
  });
}

export function useReorderGeminiKeys() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      const res = await api.put("/gemini-keys/reorder", { ids });
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.geminiKeys.all });
    },
  });
}
