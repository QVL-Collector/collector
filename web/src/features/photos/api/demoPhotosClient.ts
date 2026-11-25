import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import type { ProblemDetails } from '../../../api/problemDetails';
import type { DemoScenario } from '../../../config/runtimeConfig';
import type { ArticlePhoto, PhotoLimits, PhotosClient, PhotoUpload } from '../domain/articlePhoto';
import type { ImageDimensions } from '../domain/imageHeader';
import { inspectPhotoFile, type PhotoRejection } from '../domain/photoInspection';
import { photoHttpStatuses, photoProblemTypes } from '../domain/photoProblems';
import { createDemoPhotoBlob, demoPhotoDimensions, demoPhotoSeed } from './demoPhotoImages';

type StoredPhoto = Omit<ArticlePhoto, 'expiresAt'>;

const presignedLifetimeMs = 10 * 60 * 1000;
const uploadSteps = 8;
const uploadStepMs = 160;
const demoCatalogueArticlePrefix = 'demo-';

const seededSellerArticleIds: ReadonlySet<string> = new Set([
  '1b7e2c9a-4f3d-4c11-8a6e-2d9b0f5c7a01',
  'c2e9b4a7-1d6f-4f80-9a3b-8c5e7d2f1a03',
  '9a3d7c1e-5b2f-4a68-8e4d-1f7b6c9a2d04',
]);

const seededReviewArticleIds: ReadonlySet<string> = new Set([
  '3f8a2b6c-1d4e-4a7b-9c0d-5e6f7a8b9c21',
  'a9c3e5f7-2b4d-4e6f-8a1c-3d5e7f9b1c31',
]);

function problem(type: string): ProblemDetails {
  return { type, fieldProblems: [] };
}

function abortError(): DOMException {
  return new DOMException('Aborted', 'AbortError');
}

function rejectionError(reason: PhotoRejection): ApiError {
  switch (reason) {
    case 'unsupported-type':
      return new ApiError(
        photoHttpStatuses.unsupportedMediaType,
        problem(photoProblemTypes.unsupported),
      );
    case 'too-large':
      return new ApiError(photoHttpStatuses.payloadTooLarge, problem(photoProblemTypes.tooLarge));
    case 'too-many-pixels':
      return new ApiError(
        photoHttpStatuses.unprocessableContent,
        problem(photoProblemTypes.excessiveDimensions),
      );
  }
}

function simulateProgress({ onProgress, signal }: PhotoUpload): Promise<void> {
  return new Promise((resolve, reject) => {
    let step = 0;
    const timer = window.setInterval(() => {
      step += 1;
      onProgress(step / uploadSteps);
      if (step >= uploadSteps) {
        window.clearInterval(timer);
        resolve();
      }
    }, uploadStepMs);
    signal.addEventListener('abort', () => {
      window.clearInterval(timer);
      reject(abortError());
    });
  });
}

function withFreshExpiry(photo: StoredPhoto): ArticlePhoto {
  return { ...photo, expiresAt: new Date(Date.now() + presignedLifetimeMs) };
}

function storedPhoto(blob: Blob, { width, height }: ImageDimensions): StoredPhoto {
  return { id: crypto.randomUUID(), url: URL.createObjectURL(blob), width, height };
}

async function generatedPhotos(articleId: string): Promise<StoredPhoto[]> {
  const blob = await createDemoPhotoBlob(demoPhotoSeed(articleId));
  return blob === null ? [] : [storedPhoto(blob, demoPhotoDimensions)];
}

export function createDemoPhotosClient(scenario: DemoScenario, limits: PhotoLimits): PhotosClient {
  const photosByArticle = new Map<string, StoredPhoto[]>();

  const expectAvailable = (): void => {
    if (scenario === 'error') {
      throw new ApiError(photoHttpStatuses.serviceUnavailable);
    }
  };

  const photosOf = async (articleId: string, isSeeded: boolean): Promise<StoredPhoto[]> => {
    const stored = photosByArticle.get(articleId);
    if (stored !== undefined) {
      return stored;
    }
    const seeded = scenario === 'nominal' && isSeeded ? await generatedPhotos(articleId) : [];
    photosByArticle.set(articleId, seeded);
    return seeded;
  };

  const sellerPhotosOf = (articleId: string) =>
    photosOf(articleId, seededSellerArticleIds.has(articleId));

  const listSeeded = async (articleId: string, isSeeded: boolean, signal?: AbortSignal) => {
    await waitForDemoLatency(signal);
    expectAvailable();
    return (await photosOf(articleId, isSeeded)).map(withFreshExpiry);
  };

  return {
    limits,
    listMyArticlePhotos: (articleId, signal) =>
      listSeeded(articleId, seededSellerArticleIds.has(articleId), signal),
    async uploadPhoto(articleId, upload) {
      expectAvailable();
      const inspection = await inspectPhotoFile(upload.file, limits);
      if (inspection.kind === 'rejected') {
        throw rejectionError(inspection.reason);
      }
      const current = await sellerPhotosOf(articleId);
      if (current.length >= limits.maxPerArticle) {
        throw new ApiError(photoHttpStatuses.conflict, problem(photoProblemTypes.limitReached));
      }
      await simulateProgress(upload);
      const photo = storedPhoto(upload.file, inspection.dimensions ?? demoPhotoDimensions);
      photosByArticle.set(articleId, [...current, photo]);
      return withFreshExpiry(photo);
    },
    async deletePhoto(articleId, photoId) {
      await waitForDemoLatency();
      expectAvailable();
      const current = await sellerPhotosOf(articleId);
      const removed = current.find(({ id }) => id === photoId);
      if (removed === undefined) {
        throw new ApiError(photoHttpStatuses.notFound);
      }
      photosByArticle.set(
        articleId,
        current.filter(({ id }) => id !== photoId),
      );
      URL.revokeObjectURL(removed.url);
    },
    listPublishedArticlePhotos: (articleId, signal) =>
      listSeeded(articleId, articleId.startsWith(demoCatalogueArticlePrefix), signal),
    listReviewArticlePhotos: (articleId, signal) =>
      listSeeded(articleId, seededReviewArticleIds.has(articleId), signal),
  };
}
