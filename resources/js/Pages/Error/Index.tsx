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

interface ErrorPageProps extends SharedProps {
  status: number;
}

// Titles/descriptions are looked up client-side rather than sent as props —
// ctx.Error only passes `status` in production, deliberately not leaking
// the underlying error message to an end user.
const titles: Record<number, string> = {
  401: "Unauthorized",
  403: "Forbidden",
  404: "Page not found",
  412: "Precondition failed",
  500: "Something went wrong",
};

const descriptions: Record<number, string> = {
  401: "You need to sign in to view this page.",
  403: "You don't have permission to view this page.",
  404: "The page you're looking for doesn't exist or may have moved.",
  412: "This request can't be completed as sent.",
  500: "An unexpected error occurred. Try again in a moment.",
};

export default function ErrorPage() {
  const { status } = usePage<ErrorPageProps>().props;
  const title = titles[status] ?? "Something went wrong";
  const description = descriptions[status] ?? "An unexpected error occurred.";

  return (
    <div className="bg-muted flex min-h-screen items-center justify-center p-6">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          <p className="text-muted-foreground text-sm font-medium">
            {status}
          </p>
          <CardTitle className="text-3xl">{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild>
            <Link href="/">Go home</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
