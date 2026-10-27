export function opsPerSecond(ops, cur, prev) {
  return ops / (cur.closeTime - prev.closeTime);
}

export function checkOrder(cur, prev) {
  if (cur.ledgerCloseTime <= prev.ledgerCloseTime) {
    throw new Error("close time did not advance");
  }
}
