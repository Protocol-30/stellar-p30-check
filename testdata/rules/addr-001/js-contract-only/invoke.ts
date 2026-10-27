import { StrKey } from "@stellar/stellar-sdk";

// A contract to invoke must be a plain C address; muxed contract
// addresses cannot be invoked (CAP-84).
export function contractToInvoke(id: string): string {
  if (!id.startsWith("C") || !StrKey.isValidContract(id)) {
    throw new Error("expected a contract ID");
  }
  return id;
}
