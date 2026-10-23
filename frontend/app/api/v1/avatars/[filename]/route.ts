import { avatarImage } from "@/lib/auth/bff";

export async function GET(_request: Request, context: { params: Promise<{ filename: string }> }) {
  const { filename } = await context.params;
  return avatarImage(filename);
}
