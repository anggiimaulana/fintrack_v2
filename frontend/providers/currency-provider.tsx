"use client";

import { createContext, useContext, type ReactNode } from "react";
import { useCurrencies } from "@/hooks/useCurrencies";
import { useSession } from "@/lib/session-context";
import { BASE_CURRENCY, isPriced, setActiveCurrency, setCurrencyCatalog, type CurrencyMeta } from "@/lib/currency";

type CurrencyContextValue = {
  /** Selectable currencies (active catalog) with their current rate. */
  currencies: CurrencyMeta[];
  /** The currency the signed-in user records and sees values in. */
  displayCurrency: CurrencyMeta;
  isLoading: boolean;
};

const CurrencyContext = createContext<CurrencyContextValue>({
  currencies: [BASE_CURRENCY],
  displayCurrency: BASE_CURRENCY,
  isLoading: false,
});

/**
 * Resolves the user's display currency from the dynamic catalog and publishes it
 * both as context (for pickers) and as module state (so lib/currency formatters
 * follow it). Mounted inside SessionProvider.
 */
export function CurrencyProvider({ children }: { children: ReactNode }) {
  const { user, isAuthenticated } = useSession();
  const { data, isLoading } = useCurrencies({ enabled: isAuthenticated });

  const currencies = data && data.length > 0 ? data : [BASE_CURRENCY];
  const code = user?.display_currency || BASE_CURRENCY.code;
  const fromCatalog = currencies.find((c) => c.code === code);
  // Only treat the stored preference as active when it is selectable and priced;
  // otherwise a deactivated currency would relabel IDR amounts.
  const displayCurrency =
    fromCatalog && (fromCatalog.is_base || isPriced(fromCatalog)) ? fromCatalog : BASE_CURRENCY;

  // Publish the catalog + active currency before children render so any
  // formatMoney call in this pass sees the right currency.
  setCurrencyCatalog(currencies);
  setActiveCurrency(displayCurrency);

  return (
    <CurrencyContext.Provider value={{ currencies, displayCurrency, isLoading }}>
      {children}
    </CurrencyContext.Provider>
  );
}

export function useCurrency() {
  return useContext(CurrencyContext);
}
