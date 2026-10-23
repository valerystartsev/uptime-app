import { NextRequest, NextResponse } from "next/server";

type Credentials = {
  email: string;
  password: string;
};

type TokenResponse = {
  access_token: string;
  token_type: "Bearer";
  expires_in: number;
};

type ErrorResponse = {
  error: string;
};

type ProfileResponse = {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
};

const allowedErrorStatuses = new Set([400, 401, 409]);

export async function register(request: NextRequest) {
  return submitCredentials(request, "/api/v1/auth/register");
}

export async function login(request: NextRequest) {
  return submitCredentials(request, "/api/v1/auth/login");
}

export async function refresh(request: NextRequest) {
  return forwardSessionRequest(request, "/api/v1/auth/refresh", false);
}

export async function logout(request: NextRequest) {
  return forwardSessionRequest(request, "/api/v1/auth/logout", true);
}

export async function profile(request: NextRequest) {
  const accessToken = request.cookies.get("access_token")?.value;
  if (!accessToken) {
    return errorResponse(401, "Invalid access token");
  }

  let backendResponse: Response;
  try {
    backendResponse = await fetch(`${backendURL()}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${accessToken}` },
      cache: "no-store",
    });
  } catch {
    return errorResponse(502, "Authentication service is unavailable");
  }

  if (!backendResponse.ok) {
    return backendError(backendResponse);
  }

  const userProfile = await readProfileResponse(backendResponse);
  if (!userProfile) {
    return errorResponse(502, "Authentication service returned an invalid response");
  }
  return NextResponse.json(userProfile);
}

export async function updateProfile(request: NextRequest) {
  const accessToken = request.cookies.get("access_token")?.value;
  if (!accessToken) {
    return errorResponse(401, "Invalid access token");
  }
  let value: unknown;
  try {
    value = await request.json();
  } catch {
    return errorResponse(400, "Invalid request body");
  }
  if (!value || typeof value !== "object" || Array.isArray(value) ||
      Object.keys(value).length !== 1 || typeof (value as Record<string, unknown>).name !== "string") {
    return errorResponse(400, "Invalid request body");
  }
  const name = (value as { name: string }).name.trim();
  if (name.length === 0 || [...name].length > 100) {
    return errorResponse(400, "Name must contain 1 to 100 characters");
  }

  let backendResponse: Response;
  try {
    backendResponse = await fetch(`${backendURL()}/api/v1/auth/me`, {
      method: "PATCH",
      headers: { Authorization: `Bearer ${accessToken}`, "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
      cache: "no-store",
    });
  } catch {
    return errorResponse(502, "Authentication service is unavailable");
  }
  if (!backendResponse.ok) {
    return backendError(backendResponse);
  }
  const userProfile = await readProfileResponse(backendResponse);
  if (!userProfile) {
    return errorResponse(502, "Authentication service returned an invalid response");
  }
  return NextResponse.json(userProfile);
}

export async function uploadAvatar(request: NextRequest) {
  const token = request.cookies.get("access_token")?.value;
  if (!token) return errorResponse(401, "Invalid access token");
  if (Number(request.headers.get("content-length")) > 5 * 1024 * 1024 + 1024) {
    return errorResponse(400, "Avatar is too large");
  }
  let form: FormData;
  try { form = await request.formData(); } catch { return errorResponse(400, "Invalid avatar upload"); }
  const file = form.get("avatar");
  if (form.getAll("avatar").length !== 1 || [...form.keys()].length !== 1 || !(file instanceof File) ||
      file.size > 5 * 1024 * 1024 || !["image/png", "image/jpeg"].includes(file.type)) {
    return errorResponse(400, "Avatar must be a JPEG or PNG image up to 5 MB");
  }
  let backendResponse: Response;
  try {
    backendResponse = await fetch(`${backendURL()}/api/v1/auth/me/avatar`, {
      method: "POST", headers: { Authorization: `Bearer ${token}` }, body: form, cache: "no-store",
    });
  } catch { return errorResponse(502, "Authentication service is unavailable"); }
  if (!backendResponse.ok) return backendError(backendResponse);
  const userProfile = await readProfileResponse(backendResponse);
  return userProfile ? NextResponse.json(userProfile) : errorResponse(502, "Authentication service returned an invalid response");
}

export async function avatarImage(filename: string) {
  if (!/^[0-9a-f-]{36}\.(png|jpg)$/.test(filename)) return new Response(null, { status: 404 });
  try {
    const response = await fetch(`${backendURL()}/api/v1/avatars/${filename}`, { cache: "no-store" });
    if (!response.ok) return new Response(null, { status: response.status === 404 ? 404 : 502 });
    return new Response(response.body, { headers: {
      "Content-Type": response.headers.get("content-type") ?? "application/octet-stream",
      "X-Content-Type-Options": "nosniff",
      "Cache-Control": "no-store",
    } });
  } catch { return new Response(null, { status: 502 }); }
}

async function submitCredentials(request: NextRequest, path: string) {
  const credentials = await readCredentials(request);
  if (!credentials) {
    return errorResponse(400, "Invalid request body");
  }

  return forwardSessionRequest(request, path, false, credentials);
}

async function forwardSessionRequest(
  request: NextRequest,
  path: string,
  clearAccessToken: boolean,
  credentials?: Credentials,
) {
  let backendResponse: Response;
  try {
    backendResponse = await fetch(`${backendURL()}${path}`, {
      method: "POST",
      headers: requestHeaders(request, credentials),
      body: credentials ? JSON.stringify(credentials) : undefined,
      cache: "no-store",
    });
  } catch {
    return errorResponse(502, "Authentication service is unavailable");
  }

  if (!backendResponse.ok) {
    return backendError(backendResponse);
  }

  if (clearAccessToken) {
    const response = new NextResponse(null, { status: 204 });
    clearAccessCookie(response);
    copyRefreshCookie(backendResponse, response);
    return response;
  }

  const token = await readTokenResponse(backendResponse);
  if (!token) {
    return errorResponse(502, "Authentication service returned an invalid response");
  }

  const response = NextResponse.json({ authenticated: true });
  setAccessCookie(response, token);
  copyRefreshCookie(backendResponse, response);
  return response;
}

function backendURL() {
  const url = process.env.BACKEND_API_URL;
  if (!url) {
    throw new Error("BACKEND_API_URL is not configured");
  }
  return url.replace(/\/$/, "");
}

function requestHeaders(request: NextRequest, credentials?: Credentials) {
  const headers = new Headers();
  if (credentials) {
    headers.set("Content-Type", "application/json");
  }
  const cookie = request.headers.get("cookie");
  if (cookie) {
    headers.set("cookie", cookie);
  }
  return headers;
}

async function readCredentials(request: NextRequest): Promise<Credentials | null> {
  try {
    const value: unknown = await request.json();
    if (!isCredentials(value)) {
      return null;
    }
    return value;
  } catch {
    return null;
  }
}

function isCredentials(value: unknown): value is Credentials {
  if (!value || typeof value !== "object") {
    return false;
  }
  const credentials = value as Record<string, unknown>;
  return typeof credentials.email === "string" && typeof credentials.password === "string";
}

async function readTokenResponse(response: Response): Promise<TokenResponse | null> {
  try {
    const value: unknown = await response.json();
    if (!value || typeof value !== "object") {
      return null;
    }
    const token = value as Record<string, unknown>;
    if (
      typeof token.access_token !== "string" ||
      token.access_token.length === 0 ||
      token.token_type !== "Bearer" ||
      typeof token.expires_in !== "number" ||
      !Number.isFinite(token.expires_in) ||
      token.expires_in <= 0
    ) {
      return null;
    }
    return token as TokenResponse;
  } catch {
    return null;
  }
}

async function backendError(response: Response) {
  if (!allowedErrorStatuses.has(response.status)) {
    return errorResponse(502, "Authentication service is unavailable");
  }
  const fallback = response.status === 401 ? "Invalid credentials" : "Unable to complete the request";
  const body = await readErrorResponse(response);
  return errorResponse(response.status, body?.error ?? fallback);
}

async function readErrorResponse(response: Response): Promise<ErrorResponse | null> {
  try {
    const value: unknown = await response.json();
    if (!value || typeof value !== "object" || typeof (value as Record<string, unknown>).error !== "string") {
      return null;
    }
    return value as ErrorResponse;
  } catch {
    return null;
  }
}

async function readProfileResponse(response: Response): Promise<ProfileResponse | null> {
  try {
    const value: unknown = await response.json();
    if (!value || typeof value !== "object" ||
        typeof (value as Record<string, unknown>).id !== "string" ||
        typeof (value as Record<string, unknown>).email !== "string" ||
        typeof (value as Record<string, unknown>).name !== "string" ||
        typeof (value as Record<string, unknown>).avatar_url !== "string") {
      return null;
    }
    const profile = value as Record<string, string>;
    return { id: profile.id, email: profile.email, name: profile.name, avatar_url: profile.avatar_url };
  } catch {
    return null;
  }
}

function setAccessCookie(response: NextResponse, token: TokenResponse) {
  response.cookies.set({
    name: "access_token",
    value: token.access_token,
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: Math.floor(token.expires_in),
  });
}

function clearAccessCookie(response: NextResponse) {
  response.cookies.set({
    name: "access_token",
    value: "",
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });
}

function copyRefreshCookie(backendResponse: Response, response: NextResponse) {
	const setCookies = backendResponse.headers.getSetCookie();
	for (const setCookie of setCookies) {
		response.headers.append("set-cookie", setCookie);
	}
}

function errorResponse(status: number, error: string) {
  return NextResponse.json({ error }, { status });
}
