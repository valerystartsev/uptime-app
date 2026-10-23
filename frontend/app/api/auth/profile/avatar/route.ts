import { NextRequest } from "next/server";
import { uploadAvatar } from "@/lib/auth/bff";

export async function POST(request: NextRequest) {
  return uploadAvatar(request);
}
