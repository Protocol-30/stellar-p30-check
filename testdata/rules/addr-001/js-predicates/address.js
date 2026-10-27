import { StrKey } from "@stellar/stellar-sdk";

export function isRecipient(a) {
  return (
    StrKey.isValidEd25519PublicKey(a) ||
    StrKey.isValidMed25519PublicKey(a) ||
    StrKey.isValidContract(a)
  );
}
