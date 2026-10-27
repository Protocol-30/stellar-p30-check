// Classic payments: accounts and muxed accounts only. W is a contract
// address, so an account-only field correctly rejects it.
export const ACCOUNT_RE = /^[GM][A-Z2-7]{55}$/;

export function isAccount(a: string): boolean {
  return a.startsWith("G") || a.startsWith("M");
}
