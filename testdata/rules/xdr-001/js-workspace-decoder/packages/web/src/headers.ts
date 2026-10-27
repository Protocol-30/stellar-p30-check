import { xdr } from "@stellar/stellar-sdk";

export function closeTimeOf(headerXdr: string): bigint {
  const entry = xdr.LedgerHeaderHistoryEntry.fromXDR(headerXdr, "base64");
  return entry.header().scpValue().closeTime().toBigInt();
}
