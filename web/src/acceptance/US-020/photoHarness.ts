import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, vi } from 'vitest';
import {
  marketplace,
  type ScriptedAnswer,
  type StoredPhoto,
} from '../US-013/sellerMarketplaceHarness';

export type UploadRecord = {
  readonly path: string;
  readonly authorization: string | null;
  readonly contentType: string | null;
  readonly size: number;
};

export type PhotoUploads = {
  readonly records: UploadRecord[];
  readonly scripted: ScriptedAnswer[];
  readonly pending: (() => void)[];
  held: boolean;
};

const jpegBase64 =
  '/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AABEIAAYACAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/AKeieB7T5eU/Kug/4Qe09U/Kruifw1v14mJx2I9o/eKyTMsV9Uj77P/Z';
const pngBase64 =
  'iVBORw0KGgoAAAANSUhEUgAAAAgAAAAGCAIAAABxZ0isAAAAHUlEQVR4nGJhYEiRY2DARCwMGgxYAT0kAAEAAP//66ACF2qQ8iwAAAAASUVORK5CYII=';
const gifBase64 = 'R0lGODlhAQABAIAAAP///wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw==';
const uploadPath = /^\/marche\/v1\/vendeur\/articles\/([^/]+)\/photos$/;
const bucket = 'https://collector-photos.s3.eu-west-3.amazonaws.com/photos';

export const screenTimeout = { timeout: 15000 };
export const announcementTimeout = 15000;

let current: PhotoUploads | null = null;
let photoSequence = 0;

function bytesOf(base64: string): Uint8Array<ArrayBuffer> {
  return Uint8Array.from(atob(base64), (character) => character.charCodeAt(0));
}

export function jpegFile(name = 'objet.jpg'): File {
  return new File([bytesOf(jpegBase64)], name, { type: 'image/jpeg' });
}

export function pngFile(name = 'objet.png'): File {
  return new File([bytesOf(pngBase64)], name, { type: 'image/png' });
}

export function gifFile(): File {
  return new File([bytesOf(gifBase64)], 'anime.gif', { type: 'image/gif' });
}

export function webpFile(): File {
  return new File(['RIFF\u0010\u0000\u0000\u0000WEBPVP8L'], 'photo.webp', { type: 'image/webp' });
}

export function svgFile(): File {
  return new File(
    ['<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>'],
    'logo.svg',
    {
      type: 'image/svg+xml',
    },
  );
}

export function textNamedJpeg(): File {
  return new File(['Ceci n’est pas une image'], 'photo.jpg', { type: 'image/jpeg' });
}

export function presignedUrl(rank: number): string {
  return `${bucket}/${String(rank).padStart(26, 'a')}.jpg?X-Amz-Expires=600&X-Amz-Signature=${String(rank)}`;
}

export function storedPhoto(rank: number, expiresInMinutes = 10): StoredPhoto {
  return {
    id: `10000000-0000-4000-8000-${rank.toString().padStart(12, '0')}`,
    url: presignedUrl(rank),
    expireLe: new Date(Date.now() + expiresInMinutes * 60_000).toISOString(),
    largeur: 8,
    hauteur: 6,
  };
}

export function givenPhotos(id: string, ...photos: StoredPhoto[]): void {
  marketplace().photos.set(id, photos);
}

function photoProblem(status: number, type: string): ScriptedAnswer {
  return { status, body: { type, title: 'Problème', status } };
}

export const tooLarge = (): ScriptedAnswer =>
  photoProblem(413, 'urn:collector:probleme:photo-trop-volumineuse');
export const unsupported = (): ScriptedAnswer =>
  photoProblem(415, 'urn:collector:probleme:photo-non-prise-en-charge');
export const tooManyPixels = (): ScriptedAnswer =>
  photoProblem(422, 'urn:collector:probleme:photo-dimensions-excessives');
export const limitReached = (): ScriptedAnswer =>
  photoProblem(409, 'urn:collector:probleme:photos-limite-atteinte');
export const notModifiable = (): ScriptedAnswer =>
  photoProblem(409, 'urn:collector:probleme:article-non-modifiable');

function accept(path: string): ScriptedAnswer {
  const id = uploadPath.exec(path)?.[1] ?? '';
  if (!marketplace().articles.has(id)) {
    return photoProblem(404, 'about:blank');
  }
  photoSequence += 1;
  const photo = storedPhoto(500 + photoSequence);
  marketplace().photos.set(id, [...(marketplace().photos.get(id) ?? []), photo]);
  return { status: 201, body: photo };
}

class FakeUploadRequest extends EventTarget {
  readonly upload = new EventTarget();
  status = 0;
  responseText = '';
  private method = '';
  private path = '';
  private aborted = false;
  private answerType = '';
  private readonly headers = new Headers();

  open(method: string, url: string | URL): void {
    this.method = method;
    this.path = new URL(url, window.location.origin).pathname;
  }

  setRequestHeader(name: string, value: string): void {
    this.headers.set(name, value);
  }

  getResponseHeader(name: string): string | null {
    return name.toLowerCase() === 'content-type' ? this.answerType : null;
  }

  send(body: Blob): void {
    const uploads = photoUploads();
    uploads.records.push({
      path: this.path,
      authorization: this.headers.get('Authorization'),
      contentType: this.headers.get('Content-Type'),
      size: body.size,
    });
    const respond = (): void => {
      this.respond(body.size);
    };
    if (uploads.held) {
      uploads.pending.push(respond);
      return;
    }
    setTimeout(respond, 0);
  }

  abort(): void {
    this.aborted = true;
    this.dispatchEvent(new Event('abort'));
  }

  private respond(size: number): void {
    if (this.aborted || this.method !== 'POST') {
      return;
    }
    this.upload.dispatchEvent(
      new ProgressEvent('progress', { lengthComputable: true, loaded: size, total: size }),
    );
    const answer = photoUploads().scripted.shift() ?? accept(this.path);
    this.status = answer.status;
    this.answerType = answer.status >= 400 ? 'application/problem+json' : 'application/json';
    this.responseText = JSON.stringify(answer.body);
    this.dispatchEvent(new Event('load'));
  }
}

function readAsArrayBuffer(this: Blob): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener('load', () => {
      resolve(reader.result as ArrayBuffer);
    });
    reader.addEventListener('error', () => {
      reject(reader.error ?? new Error('Lecture du fichier impossible'));
    });
    reader.readAsArrayBuffer(this);
  });
}

beforeEach(() => {
  if (typeof Blob.prototype.arrayBuffer !== 'function') {
    Object.defineProperty(Blob.prototype, 'arrayBuffer', {
      configurable: true,
      writable: true,
      value: readAsArrayBuffer,
    });
  }
  current = { records: [], scripted: [], pending: [], held: false };
  vi.stubGlobal('XMLHttpRequest', FakeUploadRequest);
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(() => 'blob:apercu-photo'),
  });
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    writable: true,
    value: vi.fn(),
  });
});

export function photoUploads(): PhotoUploads {
  if (current === null) {
    throw new Error('Dépôt de photo simulé non initialisé');
  }
  return current;
}

export function holdUploads(): () => void {
  photoUploads().held = true;
  return () => {
    const uploads = photoUploads();
    uploads.held = false;
    for (const respond of uploads.pending.splice(0)) {
      respond();
    }
  };
}

export function photoReads(id: string): number {
  return marketplace().requests.filter(
    (request) =>
      request.method === 'GET' && request.path === `/marche/v1/vendeur/articles/${id}/photos`,
  ).length;
}

export function photoDeletions(id: string): number {
  return marketplace().requests.filter(
    (request) =>
      request.method === 'DELETE' &&
      request.path.startsWith(`/marche/v1/vendeur/articles/${id}/photos/`),
  ).length;
}

export function fileInput(name: string): HTMLInputElement {
  const input = screen.getByLabelText(name, { selector: 'input' });
  if (!(input instanceof HTMLInputElement)) {
    throw new Error('Champ de fichier introuvable');
  }
  return input;
}

export async function choosePhoto(name: string, file: File): Promise<void> {
  await userEvent.upload(fileInput(name), file, { applyAccept: false });
}
