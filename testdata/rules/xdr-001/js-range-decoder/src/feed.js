import { rpc } from "@stellar/stellar-sdk";

const server = new rpc.Server("https://soroban-testnet.stellar.org");

export async function latestLedgers(startLedger) {
  const page = await server.getLedgers({ startLedger, pagination: { limit: 10 } });
  return page.ledgers;
}
