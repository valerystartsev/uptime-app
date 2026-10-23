import { NextRequest } from "next/server";

import { refresh } from "@/lib/auth/bff";

export async function POST(request: NextRequest) {
  return refresh(request);
}
