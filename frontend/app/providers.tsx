"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { useState } from "react";
import { Toaster } from "sonner";
import { ThemeProvider } from "next-themes";
import { SessionProvider } from "@/lib/session-context";
import { RealtimeProvider } from "@/providers/realtime-provider";
import { shouldRetry } from "@/lib/query-retry";

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 1000 * 60 * 2,
            gcTime: 1000 * 60 * 30,
            refetchOnWindowFocus: true,
            // Never retry 4xx (401/402/403): the answer is definitive, and a
            // retry just duplicates the request. 5xx gets a single retry.
            retry: shouldRetry,
            refetchOnMount: true,
          },
          mutations: {
            retry: 0,
          },
        },
      })
  );

  return (
    <ThemeProvider attribute="class" defaultTheme="light" enableSystem={false} disableTransitionOnChange>
      <QueryClientProvider client={queryClient}>
        <SessionProvider>
          <RealtimeProvider>
            {children}
          </RealtimeProvider>
        </SessionProvider>
        <Toaster
          position="bottom-right"
          toastOptions={{
            style: { borderRadius: 0, border: "2px solid #000", boxShadow: "4px 4px 0px 0px #000", padding: "14px 18px", fontSize: "14px", fontWeight: 700 },
          }}
        />
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
