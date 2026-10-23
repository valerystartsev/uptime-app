import { NextRequest } from "next/server";

import { login } from "@/lib/auth/bff";

export async function POST(request: NextRequest) {
  return login(request);
}
