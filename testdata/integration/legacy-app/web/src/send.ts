const DESTINATION = /^[GMC][A-Z2-7]{55}$/;

export function validDestination(value: string): boolean {
  return DESTINATION.test(value);
}
