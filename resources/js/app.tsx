import { createInertiaApp, router, type ResolvedComponent } from "@inertiajs/react";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";
import { createRoot } from "react-dom/client";

import { syncThemeFromServer } from "@/lib/theme";
import { type SharedProps } from "@/types";

const pages = import.meta.glob<ResolvedComponent>("./Pages/**/*.tsx");

createInertiaApp({
  resolve: (name) =>
    resolvePageComponent<ResolvedComponent>(`./Pages/${name}.tsx`, pages),
  setup({ el, App, props }) {
    syncThemeFromServer((props.initialPage.props as unknown as SharedProps).theme);
    createRoot(el).render(<App {...props} />);
  },
});

// An Inertia visit swaps page props without touching <head>, so the theme
// the server rendered on the first load is re-applied from each page's
// "theme" prop (see resources/js/lib/theme.ts).
router.on("navigate", (event) => {
  syncThemeFromServer((event.detail.page.props as unknown as SharedProps).theme);
});
