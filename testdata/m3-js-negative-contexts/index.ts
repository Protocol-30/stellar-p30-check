import { StrKey } from "@stellar/stellar-sdk";

// Synthetic C-only input helpers representing CAP-84's storage-key and
// AddressObject restrictions; these are not host-runtime implementations.
export function contractStorageKey(contractId: string) {
  if (!contractId.startsWith("C")) {
    throw new Error("Storage key requires an ordinary contract");
  }
  return StrKey.decodeContract(contractId);
}

export function addressObjectContractInput(contractId: string) {
  if (!contractId.startsWith("C")) {
    throw new Error("Expected ordinary contract input");
  }
  return StrKey.decodeContract(contractId);
}

// Text must never become an executable call in the scanner's analysis.
export const documentation = "if (!address.startsWith('C')) throw 0; return StrKey.decodeMuxedContract(address);";
