import { Link, usePage } from "@inertiajs/react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { type SharedProps } from "@/types";

export default function Welcome() {
  const { auth } = usePage<SharedProps>().props;
  const user = auth?.user;

  return (
    <div className="bg-muted flex min-h-screen items-center justify-center p-6">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          <CardTitle className="text-3xl">Welcome to Spark</CardTitle>
          <CardDescription>
            A starter built on the forge framework.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4">
          {user ? (
            <>
              <p className="text-muted-foreground text-sm">
                Signed in as {user.name ?? user.email}.
              </p>
              <Button asChild>
                <Link href="/home">Go to dashboard</Link>
              </Button>
            </>
          ) : (
            <>
              <p className="text-muted-foreground text-sm">
                You are not logged in.
              </p>
              <Button asChild>
                <Link href="/login">Log in</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
