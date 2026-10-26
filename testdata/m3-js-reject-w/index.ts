import { StrKey } from "@stellar/stellar-sdk";

export function decodeDestination(address: string) {
  if (!/^[GMC][A-Z2-7]+$/.test(address)) {
    throw new Error("Unsupported address prefix");
  }
  return StrKey.decodeMuxedContract(address);
}
