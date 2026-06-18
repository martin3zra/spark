import { usePage } from "@inertiajs/react";

// forge verifies the CSRF token on every POST/PUT/PATCH/DELETE, reading it from
// the "X-CSRF-Token" header (or a csrf_token form field). The token is exposed
// as a shared Inertia prop by the server's SharedProps middleware. Spread the
// result into an Inertia request's `headers` option.
export function useCsrfHeaders() {
  const { csrf_token } = usePage().props;
  return { "X-CSRF-Token": csrf_token };
}
