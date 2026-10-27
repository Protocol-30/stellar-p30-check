import { StrKey } from "@stellar/stellar-sdk";

// Shaped like the SDK's own extractBaseAddress: the first check is one
// branch, and W is handled a few lines later in the same function.
export function baseAddress(address) {
  if (StrKey.isValidEd25519PublicKey(address) || StrKey.isValidMed25519PublicKey(address) || StrKey.isValidContract(address)) {
    return address;
  }
  if (StrKey.isValidMuxedContract(address)) {
    return StrKey.encodeContract(StrKey.decodeMuxedContract(address).subarray(0, 32));
  }
  throw new TypeError("unsupported address");
}
