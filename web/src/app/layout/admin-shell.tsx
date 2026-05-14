import { Outlet } from "react-router-dom";

export function AdminShell() {
  return (
    <main className="w-full">
      <Outlet />
    </main>
  );
}
