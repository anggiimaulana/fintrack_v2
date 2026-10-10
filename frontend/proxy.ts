import { NextResponse, type NextRequest } from "next/server";
import { jwtVerify } from "jose";

const ACCESS_COOKIE = "ft_at";
const REFRESH_COOKIE = "ft_rt";

type Role = "admin" | "user";

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
    const role = payload.role === "admin" ? "admin" : "user";
    return { role, valid: true };
  } catch {
    // Expired/invalid access token — the client will refresh if it can.
    return { role: "user", valid: false };
  }
}

export default async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const isLoginPage = pathname === "/login";
  const isPublicPage = isLoginPage || pathname === "/" || pathname === "/register";
  const isAdminRoute = pathname.startsWith("/admin");
  const isUserRoute = pathname.startsWith("/user");

  const hasRefresh = Boolean(request.cookies.get(REFRESH_COOKIE)?.value);
  const session = await readRole(request);

  // No session at all (and no refresh token to recover with) → login.
  if (!session && !hasRefresh && !isPublicPage) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Access token invalid but a refresh cookie exists → let the client refresh.
  if (session && !session.valid) {
    return NextResponse.next();
  }

  if (session?.valid && isPublicPage) {
    const dashboardPath = session.role === "admin" ? "/admin" : "/user";
    if (isLoginPage) {
      return NextResponse.redirect(new URL(dashboardPath, request.url));
    }
    return NextResponse.next();
  }

  if (session?.valid && (isAdminRoute || isUserRoute)) {
    if (isAdminRoute && session.role !== "admin") {
      return NextResponse.redirect(new URL("/user", request.url));
    }
    if (isUserRoute && session.role === "admin") {
      return NextResponse.redirect(new URL("/admin", request.url));
    }
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
