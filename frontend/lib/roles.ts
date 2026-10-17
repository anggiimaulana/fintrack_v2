export type Role = 'user' | 'admin' | 'super_admin';

export function normalizeRole(raw: unknown): Role {
  if (raw === 'super_admin') return 'super_admin';
  if (raw === 'admin') return 'admin';
  return 'user';
}

export function canAccessAdmin(role: Role): boolean {
  return role === 'admin' || role === 'super_admin';
}

export function isSuperAdmin(role: Role): boolean {
  return role === 'super_admin';
}

export function homePathForRole(role: Role): string {
  if (role === 'super_admin') return '/super-admin';
  if (role === 'admin') return '/admin';
  return '/user';
}

export type DecideRouteInput = {
  pathname: string;
  /** True when an access-token cookie was present (valid or not). */
  hasSession?: boolean;
  /** True when a refresh-token cookie was present. */
  hasRefresh?: boolean;
  role?: Role;
  valid?: boolean;
};

export type DecideRouteAction = 'redirect-login' | 'pass' | { redirect: string };

/**
 * Prefix yang boleh diakses tanpa sesi (permukaan marketing publik).
 *
 * Dipakai untuk memutuskan apakah pengunjung anonim diarahkan ke /login atau
 * dibiarkan lewat. URL berbahasa Inggris; konten tetap Bahasa Indonesia.
 */
const PUBLIC_PREFIXES = [
  '/pricing',
  '/features',
  '/about',
  '/contact',
  '/faq',
  '/privacy',
  '/terms',
] as const;

/**
 * True bila `pathname` boleh diakses tanpa sesi.
 *
 * Prefix dibandingkan sebagai **segmen penuh** (`=== p` atau `startsWith(p + '/')`),
 * bukan `startsWith(p)`, agar `/pricingx` tidak ikut lolos.
 */
export function isPublicPath(pathname: string): boolean {
  if (pathname === '/' || pathname === '/login' || pathname === '/register') return true;
  return PUBLIC_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

/**
 * Single routing decision shared by the edge proxy and its tests.
 *
 * - No access cookie: public pages pass; otherwise an existing refresh cookie
 *   is left to the client (which refreshes, or bounces to /login on failure).
 * - Invalid token but cookie present: pass, so the client can refresh.
 * - Valid session: `/super-admin` requires super_admin, `/admin` requires admin
 *   or above, `/user` requires a plain user, `/login` bounces to the role home.
 */
export function decideRoute({
  pathname,
  hasSession = false,
  hasRefresh = false,
  role = 'user',
  valid = false,
}: DecideRouteInput): DecideRouteAction {
  const publicPath = isPublicPath(pathname);

  if (!hasSession) {
    if (publicPath) return 'pass';
    return hasRefresh ? 'pass' : 'redirect-login';
  }

  if (!valid) return 'pass';

  if (publicPath) {
    return pathname === '/login' ? { redirect: homePathForRole(role) } : 'pass';
  }

  if (pathname.startsWith('/super-admin')) {
    return isSuperAdmin(role) ? 'pass' : { redirect: homePathForRole(role) };
  }

  if (pathname.startsWith('/admin')) {
    return canAccessAdmin(role) ? 'pass' : { redirect: '/user' };
  }

  if (pathname.startsWith('/user')) {
    return role === 'user' ? 'pass' : { redirect: homePathForRole(role) };
  }

  return 'pass';
}
