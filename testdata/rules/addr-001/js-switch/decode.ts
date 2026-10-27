import { xdr, StrKey } from "@stellar/stellar-sdk";

export function addressToString(addr: xdr.ScAddress): string {
  switch (addr.switch()) {
    case xdr.ScAddressType.scAddressTypeAccount():
      return StrKey.encodeEd25519PublicKey(addr.accountId().ed25519());
    case xdr.ScAddressType.scAddressTypeContract():
      return StrKey.encodeContract(addr.contractId());
    default:
      throw new Error("unsupported address type");
  }
}
