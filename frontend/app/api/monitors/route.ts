import { NextRequest } from "next/server";

import { createMonitor, listMonitors } from "@/lib/monitors/bff";

export async function GET(request: NextRequest) {
  return listMonitors(request);
}

export async function POST(request: NextRequest) {
  return createMonitor(request);
}
