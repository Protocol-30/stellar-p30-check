import { StrKey } from "@stellar/stellar-sdk";

// Intentional account-only boundary: this API does not decode W addresses.
export function decodeAccount(address: string) {
  if (!address.startsWith("G")) {
    throw new Error("Expected account address");
  }
  return StrKey.decodeEd25519PublicKey(address);
}
