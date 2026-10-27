// Destination field for payments and SAC transfers.
const DESTINATION_RE = /^[GMC][A-Z2-7]{55}$/;

export function isDestination(value: string): boolean {
  return DESTINATION_RE.test(value);
}
