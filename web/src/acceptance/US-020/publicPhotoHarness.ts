import { vi } from 'vitest';
import type { StoredPhoto } from '../US-013/sellerMarketplaceHarness';

export type PublicPhotoServer = {
  readonly reads: string[];
  answer: (id: string, read: number) => StoredPhoto[] | null;
};

const publicPhotosPath = /^\/marche\/v1\/articles\/([^/]+)\/photos$/;

function urlOf(input: RequestInfo | URL): URL {
  const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
  return new URL(href, window.location.origin);
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json' },
  });
}

export function servePublishedPhotos(
  answer: (id: string, read: number) => StoredPhoto[] | null,
): PublicPhotoServer {
  const server: PublicPhotoServer = { reads: [], answer };
  const upstream = globalThis.fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>((input, init) => {
      const match = publicPhotosPath.exec(urlOf(input).pathname);
      if (match?.[1] === undefined) {
        return upstream(input, init);
      }
      const id = decodeURIComponent(match[1]);
      server.reads.push(id);
      const photos = server.answer(id, server.reads.filter((read) => read === id).length);
      return Promise.resolve(
        photos === null
          ? json(404, { type: 'about:blank', title: 'Introuvable', status: 404 })
          : json(200, { elements: photos }),
      );
    }),
  );
  return server;
}

export function readsOf(server: PublicPhotoServer, id: string): number {
  return server.reads.filter((read) => read === id).length;
}
