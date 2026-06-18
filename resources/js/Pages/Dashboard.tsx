import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import AppLayout from "@/layouts/app-layout";
import { type BreadcrumbItem } from "@/types";

const breadcrumbs: BreadcrumbItem[] = [{ title: "Dashboard", href: "/home" }];

export default function Dashboard() {
  return (
    <AppLayout breadcrumbs={breadcrumbs}>
      <div className="flex flex-1 flex-col gap-4 p-6">
        <div>
          <h1 className="text-2xl font-semibold">Dashboard</h1>
          <p className="text-muted-foreground text-sm">
            Welcome back. This is your app shell — start building here.
          </p>
        </div>

        <div className="grid auto-rows-min gap-4 md:grid-cols-3">
          {["Overview", "Activity", "Settings"].map((title) => (
            <Card key={title}>
              <CardHeader>
                <CardTitle>{title}</CardTitle>
                <CardDescription>Placeholder card.</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="bg-muted h-24 rounded-md" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </AppLayout>
  );
}
