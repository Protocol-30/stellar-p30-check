import { StrKey } from "@stellar/stellar-sdk";

export function decodeDestination(address: string) {
  if (!address.startsWith("W")) {
    throw new Error("Expected muxed contract address");
  }
  return StrKey.decodeMuxedContract(address);
}
