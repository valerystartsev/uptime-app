"use client";

import { UserToolbar, useUserProfile } from "@/components/user-toolbar";

export function Dashboard() {
  const session = useUserProfile();

  return (
    <main className="dashboard-page">
      <UserToolbar session={session} />
    </main>
  );
}
