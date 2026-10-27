const closeTime = 5;

export function elapsed(ledger: { closeTime: number }): number {
  return Date.now() / 1000 - ledger.closeTime;
}

export function perLedger(ops: number, cur: { sequence: number }, prev: { sequence: number }): number {
  return ops / (cur.sequence - prev.sequence);
}

export function isLate(ledger: { closeTime: number }, deadline: number): boolean {
  return ledger.closeTime > deadline || closeTime > 4;
}
