import { xdr } from "@stellar/stellar-sdk";
export function decode(encoded: string) {
  return xdr.LedgerHeader.fromXDR(encoded, "base64");
}
