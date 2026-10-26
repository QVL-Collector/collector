import { ApiError } from './ApiError';
import { readProblemSummary } from './problemDetails';

export type QueryParameters = Readonly<Record<string, string>>;

export type RequestOptions = {
  readonly query?: QueryParameters;
  readonly signal?: AbortSignal;
};

export type HttpClient = {
  readonly getJson: (path: string, options?: RequestOptions) => Promise<unknown>;
};

export type HttpClientOptions = {
  readonly baseUrl: string;
  readonly getAccessToken: () => string | null;
};

function buildUrl(baseUrl: string, path: string, query: QueryParameters = {}): URL {
  const url = new URL(`${baseUrl}${path}`, window.location.origin);
  Object.entries(query).forEach(([name, value]) => {
    url.searchParams.set(name, value);
  });
  return url;
}

function buildHeaders(accessToken: string | null): Headers {
  const headers = new Headers({ Accept: 'application/json, application/problem+json' });
  if (accessToken !== null) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }
  return headers;
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError';
}

async function send(url: URL, headers: Headers, signal?: AbortSignal): Promise<Response> {
  try {
    return await fetch(url, {
      method: 'GET',
      headers,
      credentials: 'omit',
      signal: signal ?? null,
    });
  } catch (error: unknown) {
    if (isAbortError(error)) {
      throw error;
    }
    throw new ApiError('network', null, null);
  }
}

export function createHttpClient({ baseUrl, getAccessToken }: HttpClientOptions): HttpClient {
  return {
    async getJson(path, options = {}) {
      const url = buildUrl(baseUrl, path, options.query);
      const response = await send(url, buildHeaders(getAccessToken()), options.signal);
      if (!response.ok) {
        throw new ApiError('http', response.status, await readProblemSummary(response));
      }
      const payload: unknown = await response.json();
      return payload;
    },
  };
}
