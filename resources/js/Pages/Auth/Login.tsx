import { useForm } from "@inertiajs/react";
import { type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import AuthLayout from "@/layouts/auth-layout";
import { useCsrfHeaders } from "@/lib/csrf";

interface LoginProps {
  translations?: Record<string, string>;
  errors?: Record<string, string>;
}

export default function Login({ translations = {}, errors }: LoginProps) {
  const { data, setData, post, processing } = useForm({
    email: "",
    password: "",
  });
  const headers = useCsrfHeaders();
  // errors arrives as `null` (not `undefined`) on a fresh session — a
  // default parameter only kicks in for `undefined`, so it has to be
  // coerced here instead.
  const safeErrors = errors ?? {};

  function submit(e: FormEvent) {
    e.preventDefault();
    post("/login", { headers });
  }

  return (
    <AuthLayout
      title={translations["auth.login"] ?? "Log in"}
      description="Enter your credentials to access your account."
    >
      <form onSubmit={submit} className="flex flex-col gap-6">
        <div className="grid gap-2">
          <Label htmlFor="email">{translations["auth.email"] ?? "Email"}</Label>
          <Input
            id="email"
            type="email"
            value={data.email}
            onChange={(e) => setData("email", e.target.value)}
            autoComplete="email"
            autoFocus
          />
          {safeErrors.email && (
            <p className="text-destructive text-sm">{safeErrors.email}</p>
          )}
        </div>

        <div className="grid gap-2">
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

        <Button type="submit" disabled={processing} className="w-full">
          {translations["auth.login"] ?? "Log in"}
        </Button>
      </form>
    </AuthLayout>
  );
}
