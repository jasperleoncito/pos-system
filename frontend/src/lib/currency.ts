/**
 * All monetary amounts travel through the API as integer centavos.
 * These helpers convert between centavos and display strings.
 */
const phpFormatter = new Intl.NumberFormat("en-PH", {
  style: "currency",
  currency: "PHP",
});

const CENTAVOS_PER_PESO = 100;

/** Formats integer centavos as a peso string, e.g. 16000 → "₱160.00". */
export function formatCentavos(centavos: number): string {
  return phpFormatter.format(centavos / CENTAVOS_PER_PESO);
}

/** Converts a user-entered peso amount to integer centavos. */
export function pesosToCentavos(pesos: number): number {
  return Math.round(pesos * CENTAVOS_PER_PESO);
}
