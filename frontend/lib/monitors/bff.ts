import { NextRequest, NextResponse } from "next/server";

import { isIntervalUnit, isMonitor, isMonitorPage, isMonitorURL } from "@/lib/monitors/types";

const maxBodyBytes = 16 * 1024;

export async function listMonitors(request: NextRequest) {
  const cursor = request.nextUrl.searchParams.get("cursor") ?? "";
  if (!/^[A-Za-z0-9_-]{0,128}$/.test(cursor)) return error(400, "Некорректный курсор");
  return forward(request, "GET", undefined, cursor);
}

export async function createMonitor(request: NextRequest) {
  const origin = request.headers.get("origin");
  const host = request.headers.get("host");
  const protocol = request.headers.get("x-forwarded-proto") ?? request.nextUrl.protocol.slice(0, -1);
  if (!origin || !host || origin !== `${protocol}://${host}` ||
      request.headers.get("content-type")?.split(";")[0].trim().toLowerCase() !== "application/json") {
    return error(403, "Запрос отклонён");
  }
  let value: unknown;
  try { value = JSON.parse(await readLimitedBody(request)); } catch { return error(400, "Некорректный запрос"); }
  if (!value || typeof value !== "object" || Array.isArray(value)) return error(400, "Некорректный запрос");
  const body = value as Record<string, unknown>;
  if (Object.keys(body).length !== 3 || typeof body.url !== "string" || !isMonitorURL(body.url) ||
      typeof body.interval_value !== "number" || !Number.isSafeInteger(body.interval_value) ||
      body.interval_value <= 0 || !isIntervalUnit(body.interval_unit)) {
    return error(400, "Укажите URL и положительную частоту опроса");
  }
  return forward(request, "POST", body);
}

async function readLimitedBody(request: NextRequest): Promise<string> {
  if (Number(request.headers.get("content-length")) > maxBodyBytes) throw new Error("body too large");
  if (!request.body) throw new Error("empty body");
  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > maxBodyBytes) throw new Error("body too large");
      chunks.push(value);
    }
  } finally {
    void reader.cancel().catch(() => {});
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

async function forward(request: NextRequest, method: "GET" | "POST", body?: Record<string, unknown>, cursor = "") {
  const token = request.cookies.get("access_token")?.value;
  if (!token) return error(401, "Требуется вход в аккаунт");
  const backendURL = process.env.BACKEND_API_URL;
  if (!backendURL) return error(502, "Сервис недоступен");
  let response: Response;
  try {
    response = await fetch(`${backendURL.replace(/\/$/, "")}/api/v1/monitors${method === "GET" && cursor ? `?cursor=${cursor}` : ""}`, {
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
    if (response.status === 409) {
      let value: unknown;
      try { value = await response.json(); } catch { return error(502, "Сервис вернул некорректный ответ"); }
      if (value && typeof value === "object" && "error" in value) {
        if (value.error === "monitor already exists") return error(409, "Этот сайт уже добавлен");
        if (value.error === "monitor limit reached") return error(409, "Достигнут предел в 100 сайтов");
      }
      return error(502, "Сервис вернул некорректный ответ");
    }
    return error(502, "Сервис недоступен");
  }
  let value: unknown;
  try { value = await response.json(); } catch { return error(502, "Сервис вернул некорректный ответ"); }
  if (method === "GET" ? !isMonitorPage(value) : !isMonitor(value)) {
    return error(502, "Сервис вернул некорректный ответ");
  }
  return NextResponse.json(value, { status: method === "POST" ? 201 : 200 });
}

function error(status: number, message: string) {
  return NextResponse.json({ error: message }, { status });
}
