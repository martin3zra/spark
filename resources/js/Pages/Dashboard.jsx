import { router } from "@inertiajs/react";

import { useCsrfHeaders } from "@/lib/csrf";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function Dashboard({ auth = {} }) {
  const headers = useCsrfHeaders();

  return (
    <div className="bg-muted flex min-h-screen items-center justify-center p-6">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl">Dashboard</CardTitle>
          <CardDescription>
            Signed in as {auth?.user?.email ?? "unknown"}.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            variant="outline"
            onClick={() => router.post("/logout", {}, { headers })}
          >
            Log out
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
