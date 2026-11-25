import { maxPhotoSidePixels, type PhotoLimits, type PhotoMediaType } from './articlePhoto';
import { readImageDimensions, sniffPhotoMediaType, type ImageDimensions } from './imageHeader';

export type PhotoRejection = 'unsupported-type' | 'too-large' | 'too-many-pixels';

export type PhotoInspection =
  | {
      readonly kind: 'accepted';
      readonly mediaType: PhotoMediaType;
      readonly dimensions: ImageDimensions | null;
    }
  | { readonly kind: 'rejected'; readonly reason: PhotoRejection };

const inspectedHeaderBytes = 256 * 1024;

function rejected(reason: PhotoRejection): PhotoInspection {
  return { kind: 'rejected', reason };
}

function exceedsLimits({ width, height }: ImageDimensions, maxPixels: number): boolean {
  return width > maxPhotoSidePixels || height > maxPhotoSidePixels || width * height > maxPixels;
}

export function inspectPhotoHeader(bytes: Uint8Array, maxPixels: number): PhotoInspection {
  const mediaType = sniffPhotoMediaType(bytes);
  if (mediaType === null) {
    return rejected('unsupported-type');
  }
  const dimensions = readImageDimensions(bytes, mediaType);
  if (dimensions !== null && (dimensions.width === 0 || dimensions.height === 0)) {
    return rejected('unsupported-type');
  }
  if (dimensions !== null && exceedsLimits(dimensions, maxPixels)) {
    return rejected('too-many-pixels');
  }
  return { kind: 'accepted', mediaType, dimensions };
}

export async function inspectPhotoFile(file: File, limits: PhotoLimits): Promise<PhotoInspection> {
  if (file.size > limits.maxBytes) {
    return rejected('too-large');
  }
  const header = await file.slice(0, inspectedHeaderBytes).arrayBuffer();
  return inspectPhotoHeader(new Uint8Array(header), limits.maxPixels);
}
