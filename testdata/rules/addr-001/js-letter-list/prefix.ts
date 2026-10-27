const ALLOWED_PREFIXES = ["G", "M", "C"];

export const hasAllowedPrefix = (addr: string) => ALLOWED_PREFIXES.includes(addr[0]);
