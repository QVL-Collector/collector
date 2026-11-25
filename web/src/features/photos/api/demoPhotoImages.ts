import { themeVariables } from '../../../ui/theme/themeVariables';
import type { ImageDimensions } from '../domain/imageHeader';

export const demoPhotoDimensions: ImageDimensions = { width: 960, height: 720 };

const canvasWidth = demoPhotoDimensions.width;
const canvasHeight = demoPhotoDimensions.height;
const jpegQuality = 0.85;
const highlightOpacity = 0.18;
const highlightCount = 3;

const gradients: readonly (readonly [string, string])[] = [
  [themeVariables.brandPrimaryDark, themeVariables.brandPrimaryLight],
  [themeVariables.brandAccentDark, themeVariables.brandAccentLight],
  [themeVariables.brandPrimary, themeVariables.brandAccent],
];

const defaultGradient = [
  themeVariables.brandPrimaryDark,
  themeVariables.brandPrimaryLight,
] as const;

export function demoPhotoSeed(text: string): number {
  return Array.from(text).reduce((sum, character) => sum + (character.codePointAt(0) ?? 0), 0);
}

function paintGradient(context: CanvasRenderingContext2D, seed: number): void {
  const [from, to] = gradients[seed % gradients.length] ?? defaultGradient;
  const gradient = context.createLinearGradient(0, 0, canvasWidth, canvasHeight);
  gradient.addColorStop(0, from);
  gradient.addColorStop(1, to);
  context.fillStyle = gradient;
  context.fillRect(0, 0, canvasWidth, canvasHeight);
}

function paintHighlights(context: CanvasRenderingContext2D, seed: number): void {
  context.globalAlpha = highlightOpacity;
  context.fillStyle = themeVariables.surfacePaper;
  Array.from({ length: highlightCount }, (_, index) => index + 1).forEach((step) => {
    const radius = (canvasHeight / 4) * step;
    const x = ((seed * step * 97) % canvasWidth) + radius / 4;
    const y = ((seed * step * 53) % canvasHeight) + radius / 6;
    context.beginPath();
    context.arc(x, y, radius / 2, 0, Math.PI * 2);
    context.fill();
  });
  context.globalAlpha = 1;
}

export function createDemoPhotoBlob(seed: number): Promise<Blob | null> {
  const canvas = document.createElement('canvas');
  canvas.width = canvasWidth;
  canvas.height = canvasHeight;
  const context = canvas.getContext('2d');
  if (context === null) {
    return Promise.resolve(null);
  }
  paintGradient(context, seed);
  paintHighlights(context, seed);
  return new Promise((resolve) => {
    canvas.toBlob(resolve, 'image/jpeg', jpegQuality);
  });
}
