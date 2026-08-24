import { Link, usePage } from "@inertiajs/react";
import { type PropsWithChildren } from "react";

import { type SharedProps } from "@/types";

export default function AuthSimpleLayout({
  children,
  title,
  description,
}: PropsWithChildren<{ title: string; description?: string }>) {
  const { appName } = usePage<SharedProps>().props;

  return (
    <div className="bg-muted flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
      <div className="flex w-full max-w-sm flex-col gap-6">
        <Link href="/" className="flex flex-col items-center gap-2 font-medium">
          <div className="bg-primary text-primary-foreground flex size-9 items-center justify-center rounded-md">
            <span className="font-bold">{appName.charAt(0)}</span>
          </div>
          <span className="sr-only">{appName}</span>
        </Link>

        <div className="flex flex-col items-center gap-2 text-center">
          <h1 className="text-xl font-medium">{title}</h1>
          {description && (
            <p className="text-muted-foreground text-center text-sm">
              {description}
            </p>
          )}
        </div>

        {children}
      </div>
    </div>
  );
}
