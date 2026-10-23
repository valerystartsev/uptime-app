import { NextRequest, NextResponse } from "next/server";

import { isIntervalUnit, isMonitor, isMonitorURL } from "@/lib/monitors/types";

export async function listMonitors(request: NextRequest) {
  return forward(request, "GET");
}

export async function createMonitor(request: NextRequest) {
  let value: unknown;
  try { value = await request.json(); } catch { return error(400, "Некорректный запрос"); }
  if (!value || typeof value !== "object" || Array.isArray(value)) return error(400, "Некорректный запрос");
  const body = value as Record<string, unknown>;
  if (Object.keys(body).length !== 3 || typeof body.url !== "string" || !isMonitorURL(body.url) ||
      typeof body.interval_value !== "number" || !Number.isSafeInteger(body.interval_value) ||
      body.interval_value <= 0 || !isIntervalUnit(body.interval_unit)) {
    return error(400, "Укажите URL и положительную частоту опроса");
  }
  return forward(request, "POST", body);
}

async function forward(request: NextRequest, method: "GET" | "POST", body?: Record<string, unknown>) {
  const token = request.cookies.get("access_token")?.value;
  if (!token) return error(401, "Требуется вход в аккаунт");
  const backendURL = process.env.BACKEND_API_URL;
  if (!backendURL) return error(502, "Сервис недоступен");
  let response: Response;
  try {
    response = await fetch(`${backendURL.replace(/\/$/, "")}/api/v1/monitors`, {
      method,
      headers: { Authorization: `Bearer ${token}`, ...(body ? { "Content-Type": "application/json" } : {}) },
      body: body ? JSON.stringify(body) : undefined,
      cache: "no-store",
    });
  } catch {
    return error(502, "Сервис недоступен");
  }
  if (!response.ok) {
    if (response.status === 401) return error(401, "Требуется вход в аккаунт");
    if (response.status === 400) return error(400, "Проверьте URL и частоту опроса");
    return error(502, "Сервис недоступен");
  }
  let value: unknown;
  try { value = await response.json(); } catch { return error(502, "Сервис вернул некорректный ответ"); }
  if (method === "GET" ? !Array.isArray(value) || !value.every(isMonitor) : !isMonitor(value)) {
    return error(502, "Сервис вернул некорректный ответ");
  }
  return NextResponse.json(value, { status: method === "POST" ? 201 : 200 });
}

function error(status: number, message: string) {
  return NextResponse.json({ error: message }, { status });
}
