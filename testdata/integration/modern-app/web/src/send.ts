import { StrKey } from "@stellar/stellar-sdk";

export function validDestination(value: string): boolean {
  return (
    StrKey.isValidEd25519PublicKey(value) ||
    StrKey.isValidMed25519PublicKey(value) ||
    StrKey.isValidContract(value) ||
    StrKey.isValidMuxedContract(value)
  );
}
