import { NextResponse, type NextRequest } from "next/server";
import { jwtVerify } from "jose";
import { decideRoute, normalizeRole, type Role } from "@/lib/roles";

const ACCESS_COOKIE = "ft_at";
const REFRESH_COOKIE = "ft_rt";

async function readRole(request: NextRequest): Promise<{ role: Role; valid: boolean } | null> {
  const token = request.cookies.get(ACCESS_COOKIE)?.value;
  if (!token) return null;

  const secret = process.env.JWT_SECRET;
  if (!secret) return null;

  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(secret), {
      issuer: "fintrack",
      audience: "fintrack-web",
    });
    return { role: normalizeRole(payload.role), valid: true };
  } catch {
    // Expired/invalid access token — the client will refresh if it can.
    return { role: "user", valid: false };
  }
}

export default async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasRefresh = Boolean(request.cookies.get(REFRESH_COOKIE)?.value);
  const session = await readRole(request);

  const action = decideRoute({
    pathname,
    hasSession: session !== null,
    hasRefresh,
    role: session?.role,
    valid: session?.valid,
  });

  if (action === "redirect-login") {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  if (action !== "pass") {
    return NextResponse.redirect(new URL(action.redirect, request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Exclude Next internals, static files, and the /api proxy (which must
    // reach the backend rewrites unauthenticated-by-middleware).
    "/((?!api|_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)",
  ],
};
