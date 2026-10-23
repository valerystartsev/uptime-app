import { NextRequest } from "next/server";

import { logout } from "@/lib/auth/bff";

export async function POST(request: NextRequest) {
  return logout(request);
}
