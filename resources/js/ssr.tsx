import { createInertiaApp, type ResolvedComponent } from "@inertiajs/react";
import createServer from "@inertiajs/react/server";
import { resolvePageComponent } from "laravel-vite-plugin/inertia-helpers";
import ReactDOMServer from "react-dom/server";

const pages = import.meta.glob<ResolvedComponent>("./Pages/**/*.tsx");

createServer((page) =>
  createInertiaApp({
    page,
    render: ReactDOMServer.renderToString,
    resolve: (name) =>
      resolvePageComponent<ResolvedComponent>(`./Pages/${name}.tsx`, pages),
    setup: ({ App, props }) => <App {...props} />,
  }),
);
