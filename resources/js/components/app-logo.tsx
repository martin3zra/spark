import { usePage } from "@inertiajs/react";

import { type SharedProps } from "@/types";

export default function AppLogo() {
  const { appName } = usePage<SharedProps>().props;

  return (
    <>
      <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-md">
        <span className="text-sm font-bold">{appName.charAt(0)}</span>
      </div>
      <div className="ml-1 grid flex-1 text-left text-sm">
        <span className="mb-0.5 truncate leading-tight font-semibold">{appName}</span>
      </div>
    </>
  );
}
