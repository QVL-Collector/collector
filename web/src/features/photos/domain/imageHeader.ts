import type { PhotoMediaType } from './articlePhoto';

export type ImageDimensions = {
  readonly width: number;
  readonly height: number;
};

const jpegSignature = [0xff, 0xd8, 0xff] as const;
const pngSignature = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a] as const;

const pngWidthOffset = 16;
const pngHeightOffset = 20;
const pngHeaderLength = 24;

const markerPrefix = 0xff;
const firstSegmentOffset = 2;
const frameHeightOffset = 5;
const frameWidthOffset = 7;
const frameHeaderLength = 9;
const standaloneMarkers = new Set([0x01, 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8]);
const startOfFrameMarkers = new Set([
  0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf,
]);

function startsWith(bytes: Uint8Array, signature: readonly number[]): boolean {
  return signature.every((value, index) => bytes[index] === value);
}

export function sniffPhotoMediaType(bytes: Uint8Array): PhotoMediaType | null {
  if (startsWith(bytes, pngSignature)) {
    return 'image/png';
  }
  return startsWith(bytes, jpegSignature) ? 'image/jpeg' : null;
}

function readPngDimensions(view: DataView): ImageDimensions | null {
  if (view.byteLength < pngHeaderLength) {
    return null;
  }
  return { width: view.getUint32(pngWidthOffset), height: view.getUint32(pngHeightOffset) };
}

function readJpegDimensions(view: DataView): ImageDimensions | null {
  let offset = firstSegmentOffset;
  while (offset + frameHeaderLength <= view.byteLength) {
    if (view.getUint8(offset) !== markerPrefix) {
      return null;
    }
    const marker = view.getUint8(offset + 1);
    if (marker === markerPrefix) {
      offset += 1;
    } else if (standaloneMarkers.has(marker)) {
      offset += 2;
    } else if (startOfFrameMarkers.has(marker)) {
      return {
        width: view.getUint16(offset + frameWidthOffset),
        height: view.getUint16(offset + frameHeightOffset),
      };
    } else {
      offset += 2 + view.getUint16(offset + 2);
    }
  }
  return null;
}

export function readImageDimensions(
  bytes: Uint8Array,
  mediaType: PhotoMediaType,
): ImageDimensions | null {
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  switch (mediaType) {
    case 'image/png':
      return readPngDimensions(view);
    case 'image/jpeg':
      return readJpegDimensions(view);
  }
}
