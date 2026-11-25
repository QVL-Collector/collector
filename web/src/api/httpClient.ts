import { ApiError } from './ApiError';
import { noProblemDetails, parseProblemDetails, type ProblemDetails } from './problemDetails';
import { sendUpload, type UploadOptions, type UploadResponse } from './uploadRequest';

export type QueryParameters = Readonly<Record<string, string>>;

export type RequestOptions = {
  readonly query?: QueryParameters;
  readonly signal?: AbortSignal;
};

export type JsonMethod = 'POST' | 'PUT';

export type HttpClient = {
  readonly getJson: (path: string, options?: RequestOptions) => Promise<unknown>;
  readonly sendJson: (method: JsonMethod, path: string, body: unknown) => Promise<unknown>;
  readonly sendBinary: (
    path: string,
    body: Blob,
    mediaType: string,
    options?: UploadOptions,
  ) => Promise<unknown>;
  readonly deleteResource: (path: string) => Promise<void>;
};

export type HttpClientOptions = {
  readonly baseUrl: string;
  readonly getAccessToken: () => string | null;
};

const jsonMediaType = 'application/json';
const problemMediaType = 'application/problem+json';
const successStatuses = { min: 200, max: 299 } as const;

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

export function isAbortError(error: unknown): boolean {
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

function parseProblemBody(contentType: string, body: string): ProblemDetails {
  if (!contentType.includes(problemMediaType)) {
    return noProblemDetails;
  }
  try {
    const payload: unknown = JSON.parse(body);
    return parseProblemDetails(payload);
  } catch {
    return noProblemDetails;
  }
}

async function readProblem(response: Response): Promise<ProblemDetails> {
  const body = await response.text().catch(() => '');
  return parseProblemBody(response.headers.get('Content-Type') ?? '', body);
}

async function readJson(response: Response): Promise<unknown> {
  if (!response.ok) {
    throw new ApiError(response.status, await readProblem(response));
  }
  const payload: unknown = await response.json();
  return payload;
}

function isSuccess(status: number): boolean {
  return status >= successStatuses.min && status <= successStatuses.max;
}

function readUpload({ status, contentType, body }: UploadResponse): unknown {
  if (!isSuccess(status)) {
    throw new ApiError(status, parseProblemBody(contentType, body));
  }
  const payload: unknown = JSON.parse(body);
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
    async sendBinary(path, body, mediaType, options = {}) {
      const headers = buildHeaders(getAccessToken());
      headers.set('Content-Type', mediaType);
      const response = await sendUpload({
        url: buildUrl(baseUrl, path),
        headers,
        body,
        ...options,
      });
      return readUpload(response);
    },
    async deleteResource(path) {
      const response = await send(buildUrl(baseUrl, path), {
        method: 'DELETE',
        headers: buildHeaders(getAccessToken()),
      });
      if (!response.ok) {
        throw new ApiError(response.status, await readProblem(response));
      }
    },
  };
}
