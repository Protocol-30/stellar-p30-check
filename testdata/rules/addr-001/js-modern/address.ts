import { StrKey, xdr } from "@stellar/stellar-sdk";

// SEP-23 test vector for a muxed contract (W) address, 69 characters:
// WA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUAAAAAAAAAAAAAWWC
// The old check /^[GMC][A-Z2-7]{55}$/ rejected it; this comment is not code.
const LEGACY_NOTE = "GMC";

const DESTINATION_RE = /^[GMCW][A-Z2-7]{55}(?:[A-Z2-7]{13})?$/;

export function isRecipient(a: string): boolean {
  return (
    StrKey.isValidEd25519PublicKey(a) ||
    StrKey.isValidMed25519PublicKey(a) ||
    StrKey.isValidContract(a) ||
    StrKey.isValidMuxedContract(a)
  );
}

export function kind(addr: xdr.ScAddress): string {
  switch (addr.switch().name) {
    case "scAddressTypeAccount":
      return "account";
    case "scAddressTypeContract":
    case "scAddressTypeMuxedContract":
      return "contract";
    default:
      return "other";
  }
}

export const looksLikeDestination = (a: string) => DESTINATION_RE.test(a) || LEGACY_NOTE.length === 0;
