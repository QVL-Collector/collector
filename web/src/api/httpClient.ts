import { ApiError } from './ApiError';
import { noProblemDetails, parseProblemDetails, type ProblemDetails } from './problemDetails';

export type QueryParameters = Readonly<Record<string, string>>;

export type RequestOptions = {
  readonly query?: QueryParameters;
  readonly signal?: AbortSignal;
};

export type JsonMethod = 'POST' | 'PUT';

export type HttpClient = {
  readonly getJson: (path: string, options?: RequestOptions) => Promise<unknown>;
  readonly sendJson: (method: JsonMethod, path: string, body: unknown) => Promise<unknown>;
};

export type HttpClientOptions = {
  readonly baseUrl: string;
  readonly getAccessToken: () => string | null;
};

const jsonMediaType = 'application/json';
const problemMediaType = 'application/problem+json';

function buildUrl(baseUrl: string, path: string, query: QueryParameters = {}): URL {
  const url = new URL(`${baseUrl}${path}`, window.location.origin);
  Object.entries(query).forEach(([name, value]) => {
    url.searchParams.set(name, value);
  });
  return url;
}

function buildHeaders(accessToken: string | null): Headers {
  const headers = new Headers({ Accept: `${jsonMediaType}, ${problemMediaType}` });
  if (accessToken !== null) {
    headers.set('Authorization', `Bearer ${accessToken}`);
  }
  return headers;
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError';
}

async function send(url: URL, init: RequestInit): Promise<Response> {
  try {
    return await fetch(url, { ...init, credentials: 'omit' });
  } catch (error: unknown) {
    if (isAbortError(error)) {
      throw error;
    }
    throw new ApiError(null);
  }
}

async function readProblem(response: Response): Promise<ProblemDetails> {
  const contentType = response.headers.get('Content-Type') ?? '';
  if (!contentType.includes(problemMediaType)) {
    return noProblemDetails;
  }
  try {
    const payload: unknown = await response.json();
    return parseProblemDetails(payload);
  } catch {
    return noProblemDetails;
  }
}

async function readJson(response: Response): Promise<unknown> {
  if (!response.ok) {
    throw new ApiError(response.status, await readProblem(response));
  }
  const payload: unknown = await response.json();
  return payload;
}

export function createHttpClient({ baseUrl, getAccessToken }: HttpClientOptions): HttpClient {
  return {
    async getJson(path, options = {}) {
      const response = await send(buildUrl(baseUrl, path, options.query), {
        method: 'GET',
        headers: buildHeaders(getAccessToken()),
        signal: options.signal ?? null,
      });
      return readJson(response);
    },
    async sendJson(method, path, body) {
      const headers = buildHeaders(getAccessToken());
      headers.set('Content-Type', jsonMediaType);
      const response = await send(buildUrl(baseUrl, path), {
        method,
        headers,
        body: JSON.stringify(body),
      });
      return readJson(response);
    },
  };
}
