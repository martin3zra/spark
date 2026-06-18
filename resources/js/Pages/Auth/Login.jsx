import { useForm } from "@inertiajs/react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function Login({ translations = {}, errors = {} }) {
  const { data, setData, post, processing } = useForm({
    email: "",
    password: "",
  });

  function submit(e) {
    e.preventDefault();
    post("/login");
  }

  return (
    <div className="bg-muted flex min-h-screen items-center justify-center p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-2xl">
            {translations["auth.login"] ?? "Log in"}
          </CardTitle>
          <CardDescription>
            Enter your credentials to access your account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="email">
                {translations["auth.email"] ?? "Email"}
              </Label>
              <Input
                id="email"
                type="email"
                value={data.email}
                onChange={(e) => setData("email", e.target.value)}
                autoComplete="email"
              />
              {errors?.email && (
                <p className="text-destructive text-sm">{errors.email}</p>
              )}
            </div>
            <div className="flex flex-col gap-2">
              <Label htmlFor="password">
                {translations["auth.password"] ?? "Password"}
              </Label>
              <Input
                id="password"
                type="password"
                value={data.password}
                onChange={(e) => setData("password", e.target.value)}
                autoComplete="current-password"
              />
            </div>
            <Button type="submit" disabled={processing} className="mt-2 w-full">
              {translations["auth.login"] ?? "Log in"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
