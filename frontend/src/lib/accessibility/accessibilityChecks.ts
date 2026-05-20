export interface RGBColor {
  r: number;
  g: number;
  b: number;
}

export interface SearchShellLayout {
  viewportWidth: number;
  columns: number;
  sidebarPlacement: 'top' | 'left';
  resultColumns: number;
}

export const searchShellFocusOrder = [
  'sidebar-toggle',
  'mode-single',
  'mode-replacement',
  'mode-diet',
  'macro-protein',
  'macro-carbs',
  'macro-fat',
  'search-input',
  'autocomplete-option',
  'result-card',
  'pagination',
  'settings'
] as const;

export function parseHexColor(hex: string): RGBColor {
  const normalized = hex.trim().replace(/^#/, '');
  if (!/^[\da-fA-F]{6}$/.test(normalized)) {
    throw new Error(`invalid hex color: ${hex}`);
  }
  return {
    r: Number.parseInt(normalized.slice(0, 2), 16),
    g: Number.parseInt(normalized.slice(2, 4), 16),
    b: Number.parseInt(normalized.slice(4, 6), 16)
  };
}

export function contrastRatio(foreground: string, background: string): number {
  const fg = relativeLuminance(parseHexColor(foreground));
  const bg = relativeLuminance(parseHexColor(background));
  const lighter = Math.max(fg, bg);
  const darker = Math.min(fg, bg);
  return (lighter + 0.05) / (darker + 0.05);
}

export function passesWcagAA(foreground: string, background: string, largeText = false): boolean {
  return contrastRatio(foreground, background) >= (largeText ? 3 : 4.5);
}

export function searchShellLayout(viewportWidth: number): SearchShellLayout {
  const desktop = viewportWidth >= 640;
  return {
    viewportWidth,
    columns: desktop ? 12 : 1,
    sidebarPlacement: desktop ? 'left' : 'top',
    resultColumns: viewportWidth >= 1024 ? 3 : viewportWidth >= 640 ? 2 : 1
  };
}

export function wrapsWithinLines(text: string, maxCharsPerLine: number, maxLines: number): boolean {
  if (maxCharsPerLine <= 0 || maxLines <= 0) {
    return false;
  }
  const longestToken = text.split(/\s+/).reduce((longest, token) => Math.max(longest, token.length), 0);
  return longestToken <= maxCharsPerLine && Math.ceil(text.length / maxCharsPerLine) <= maxLines;
}

function relativeLuminance(color: RGBColor): number {
  const channels = [color.r, color.g, color.b].map((value) => {
    const normalized = value / 255;
    return normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
}
