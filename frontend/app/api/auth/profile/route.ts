import { NextRequest } from "next/server";

import { profile, updateProfile } from "@/lib/auth/bff";

export async function GET(request: NextRequest) {
  return profile(request);
}

export async function PATCH(request: NextRequest) {
  return updateProfile(request);
}
