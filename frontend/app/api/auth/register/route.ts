import { NextRequest } from "next/server";

import { register } from "@/lib/auth/bff";

export async function POST(request: NextRequest) {
  return register(request);
}
