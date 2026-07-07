/** Tenant branding → CSS custom properties, with contrast-aware foregrounds. */

const HEX_PATTERN = /^#?([0-9a-f]{6})$/i;

function hexToRgb(hex: string): [number, number, number] | null {
  const match = HEX_PATTERN.exec(hex.trim());
  if (!match) return null;
  const value = parseInt(match[1], 16);
  return [(value >> 16) & 255, (value >> 8) & 255, value & 255];
}

/** WCAG relative luminance. */
function luminance([r, g, b]: [number, number, number]): number {
  const channel = (c: number) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

/** Picks a readable foreground (near-white or near-black) for a background. */
export function contrastForeground(hex: string): string {
  const rgb = hexToRgb(hex);
  if (!rgb) return "#fafafa";
  return luminance(rgb) > 0.4 ? "#1c1917" : "#fafafa";
}

/**
 * Builds the inline CSS variables that re-theme the dashboard with a
 * tenant's brand colors. Applied on the tenant layout root so every
 * Tailwind token beneath it resolves to the brand palette.
 */
export function tenantThemeVars(colors: {
  primary: string;
  secondary: string;
  accent: string;
}): Record<string, string> {
  return {
    "--primary": colors.primary,
    "--primary-foreground": contrastForeground(colors.primary),
    "--accent": colors.accent,
    "--accent-foreground": contrastForeground(colors.accent),
    "--ring": colors.primary,
    "--sidebar-primary": colors.primary,
    "--sidebar-primary-foreground": contrastForeground(colors.primary),
    "--sidebar-ring": colors.primary,
    "--chart-1": colors.primary,
    "--chart-2": colors.accent,
    "--chart-3": colors.secondary,
  };
}
