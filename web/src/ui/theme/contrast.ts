const hexColorPattern = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i;
const channelMaximum = 255;
const luminanceOffset = 0.05;

function toLinearChannel(channel: number): number {
  const normalized = channel / channelMaximum;
  return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
}

function parseHexColor(color: string): readonly [number, number, number] {
  const match = hexColorPattern.exec(color);
  if (match === null) {
    throw new Error(`Unsupported color ${color}`);
  }
  const [, red = '', green = '', blue = ''] = match;
  return [Number.parseInt(red, 16), Number.parseInt(green, 16), Number.parseInt(blue, 16)];
}

export function relativeLuminance(color: string): number {
  const [red, green, blue] = parseHexColor(color);
  return (
    0.2126 * toLinearChannel(red) + 0.7152 * toLinearChannel(green) + 0.0722 * toLinearChannel(blue)
  );
}

export function contrastRatio(foreground: string, background: string): number {
  const luminances = [relativeLuminance(foreground), relativeLuminance(background)];
  return (Math.max(...luminances) + luminanceOffset) / (Math.min(...luminances) + luminanceOffset);
}
