// Accepts an account or a contract.
export const isAccountOrContract = (v: string) => new RegExp("^[GC][A-Z2-7]{55}$").test(v);
