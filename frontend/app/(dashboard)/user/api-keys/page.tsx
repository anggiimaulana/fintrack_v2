"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

/**
 * DEPRECATED (P03): the per-user BYOK API-key page has been removed. FinTrack
 * now uses a centralized AI gateway, so this route redirects to settings.
 */
export default function ApiKeysDeprecatedPage() {
  const router = useRouter();
  useEffect(() => {
    router.replace("/user/subscription");
  }, [router]);

  return (
    <div className="flex min-h-[40vh] items-center justify-center">
      <p className="text-sm font-bold text-muted-foreground">
        Menu API Keys telah dihapus. Mengalihkan ke halaman Langganan...
      </p>
    </div>
  );
}
