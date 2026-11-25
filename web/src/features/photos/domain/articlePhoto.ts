export const photoMediaTypes = ['image/jpeg', 'image/png'] as const;

export type PhotoMediaType = (typeof photoMediaTypes)[number];

export type ArticlePhoto = {
  readonly id: string;
  readonly url: string;
  readonly expiresAt: Date;
  readonly width: number;
  readonly height: number;
};

export type PhotoUpload = {
  readonly file: File;
  readonly mediaType: PhotoMediaType;
  readonly onProgress: (ratio: number) => void;
  readonly signal: AbortSignal;
};

export type PhotoLimits = {
  readonly maxBytes: number;
  readonly maxPixels: number;
  readonly maxPerArticle: number;
};

export type PhotosClient = {
  readonly limits: PhotoLimits;
  readonly listMyArticlePhotos: (
    articleId: string,
    signal?: AbortSignal,
  ) => Promise<readonly ArticlePhoto[]>;
  readonly uploadPhoto: (articleId: string, upload: PhotoUpload) => Promise<ArticlePhoto>;
  readonly deletePhoto: (articleId: string, photoId: string) => Promise<void>;
  readonly listPublishedArticlePhotos: (
    articleId: string,
    signal?: AbortSignal,
  ) => Promise<readonly ArticlePhoto[]>;
  readonly listReviewArticlePhotos: (
    articleId: string,
    signal?: AbortSignal,
  ) => Promise<readonly ArticlePhoto[]>;
};

export const bytesPerMebibyte = 1024 * 1024;

export const pixelsPerMegapixel = 1_000_000;

export const maxPhotoSidePixels = 65_535;

export const maxPhotosPerArticleCeiling = 10;

export const defaultPhotoLimits: PhotoLimits = {
  maxBytes: 5 * bytesPerMebibyte,
  maxPixels: 16 * pixelsPerMegapixel,
  maxPerArticle: 1,
};
