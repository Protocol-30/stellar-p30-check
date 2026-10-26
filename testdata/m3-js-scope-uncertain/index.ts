import { StrKey } from "@stellar/stellar-sdk";

export function decodeDestination(address: string) {
  if (!address.startsWith("C")) {
    throw new Error("Unsupported address prefix");
  }
  return StrKey.decodeMuxedContract(address);
}
