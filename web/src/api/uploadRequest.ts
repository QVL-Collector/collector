import { ApiError } from './ApiError';

export type UploadOptions = {
  readonly onProgress?: (ratio: number) => void;
  readonly signal?: AbortSignal;
};

export type UploadRequest = UploadOptions & {
  readonly url: URL;
  readonly headers: Headers;
  readonly body: Blob;
};

export type UploadResponse = {
  readonly status: number;
  readonly contentType: string;
  readonly body: string;
};

function abortError(): DOMException {
  return new DOMException('Aborted', 'AbortError');
}

export function sendUpload({
  url,
  headers,
  body,
  onProgress,
  signal,
}: UploadRequest): Promise<UploadResponse> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted === true) {
      reject(abortError());
      return;
    }
    const request = new XMLHttpRequest();
    request.open('POST', url);
    headers.forEach((value, name) => {
      request.setRequestHeader(name, value);
    });
    request.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable && event.total > 0) {
        onProgress?.(event.loaded / event.total);
      }
    });
    request.addEventListener('load', () => {
      resolve({
        status: request.status,
        contentType: request.getResponseHeader('Content-Type') ?? '',
        body: request.responseText,
      });
    });
    request.addEventListener('error', () => {
      reject(new ApiError(null));
    });
    request.addEventListener('abort', () => {
      reject(abortError());
    });
    signal?.addEventListener('abort', () => {
      request.abort();
    });
    request.send(body);
  });
}
