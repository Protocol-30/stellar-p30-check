import { xdr } from "@stellar/stellar-sdk";

export function arm(value: xdr.StellarValue): string {
  switch (value.ext().switch().name) {
    case "stellarValueSigned":
      return "signed";
    case "stellarValueEmptyTxSet":
      return "empty";
    default:
      return "basic";
  }
}
