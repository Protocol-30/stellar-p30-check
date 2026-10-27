import { StrKey } from "@stellar/stellar-sdk";

export class Wallet {
  isRecipient(a: string): boolean {
    return StrKey.isValidEd25519PublicKey(a) || StrKey.isValidMed25519PublicKey(a) || StrKey.isValidContract(a);
  }

  // W handling in another method does not cover isRecipient.
  isMuxedContract(a: string): boolean {
    return StrKey.isValidMuxedContract(a);
  }
}
