import { createRouter, type RouteRecordRaw, type RouterHistory, type Router } from 'vue-router'

import { routePath, routeTable } from '../lib/routeTable'

/** views maps each routes.json entry's name to the view component that
 *  renders it: a frontend-only concern routes.json has no use for.
 *  router/index.test.ts checks that every name routes.json carries has
 *  an entry here; a name with none falls back to the 404 view below
 *  rather than crashing the app. */
export const views: Record<string, () => Promise<unknown>> = {
  home: () => import('../views/HomeView.vue'),
  'imprint-de': () => import('../views/ImprintView.vue'),
  'imprint-en': () => import('../views/ImprintView.vue'),
  'privacy-de': () => import('../views/PrivacyView.vue'),
  'privacy-en': () => import('../views/PrivacyView.vue'),
}

const notFoundView = () => import('../views/NotFoundView.vue')

/** NOT_FOUND_ROUTE_NAME is the 404 page's own router name: it answers no
 *  path of its own, so it is deliberately not part of routes.json (see
 *  internal/routes/routes.go) and is the one route this file still
 *  writes by hand instead of deriving from the table. lib/seo.ts has its
 *  own equivalent, NOT_FOUND. */
export const NOT_FOUND_ROUTE_NAME = 'not-found'

/** pageRoutes builds one vue-router route per routes.json entry: adding a
 *  page there is now enough to route to it here too, as long as a
 *  matching entry also exists in views above (router/index.test.ts is
 *  the check for that). Exported for that test. */
export const pageRoutes: RouteRecordRaw[] = routeTable.map((entry) => ({
  path: routePath(entry),
  name: entry.name,
  component: views[entry.name] ?? notFoundView,
}))

const unavailableView = () => import('../views/UnavailableView.vue')

/** UNAVAILABLE_ROUTE_NAME is the placeholder the Go service answers with
 *  status 503 when a page that reads from cvd has no data yet. Like the
 *  404 page it is not part of routes.json: the server picks it by status,
 *  never by path, so the path below only lets the router render it. */
export const UNAVAILABLE_ROUTE_NAME = 'unavailable'

const routes: RouteRecordRaw[] = [
  ...pageRoutes,
  {
    path: '/unavailable/',
    name: UNAVAILABLE_ROUTE_NAME,
    component: unavailableView,
  },
  {
    path: '/:pathMatch(.*)*',
    name: NOT_FOUND_ROUTE_NAME,
    component: notFoundView,
  },
]

/** createAppRouter builds the router over history, so the client supplies
 *  a browser history and the prerender script supplies a memory one. */
export function createAppRouter(history: RouterHistory): Router {
  return createRouter({ history, routes })
}
