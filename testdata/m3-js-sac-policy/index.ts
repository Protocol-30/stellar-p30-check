import { TransactionBuilder } from "@stellar/stellar-sdk";

declare const source: ConstructorParameters<typeof TransactionBuilder>[0];
declare const options: ConstructorParameters<typeof TransactionBuilder>[1];
declare const asset: Parameters<TransactionBuilder["addSacTransferOperation"]>[1];

export function transferToAccount(destination: string) {
  if (!destination.startsWith("G")) {
    throw new Error("Application policy permits accounts only");
  }
  return new TransactionBuilder(source, options).addSacTransferOperation(destination, asset, "1");
}
