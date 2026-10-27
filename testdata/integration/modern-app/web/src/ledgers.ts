import { rpc } from "@stellar/stellar-sdk";

export async function recent(server: rpc.Server, startLedger: number) {
  const { ledgers } = await server.getLedgers({ startLedger });
  return ledgers;
}
