import { router } from "@inertiajs/react";

export default function Dashboard({ auth = {} }) {
  return (
    <div style={{ maxWidth: 640, margin: "4rem auto" }}>
      <h1>Dashboard</h1>
      <p>Signed in as {auth?.user?.email ?? "unknown"}.</p>
      <button onClick={() => router.post("/logout")}>Log out</button>
    </div>
  );
}
