package rules

import (
	"path"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/walker"
)

func jsAddressInput(source, extension string) walker.File {
	kind := "js_source"
	if extension == ".ts" || extension == ".tsx" {
		kind = "ts_source"
	}
	return walker.File{Path: "src/sample" + extension, Kind: kind, Size: int64(len(source)), Data: []byte(source)}
}

func TestJSAddressWOnlyCandidates(t *testing.T) {
	tests := []struct {
		name, extension, source, sink string
	}{
		{"named-import", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) {
  if (!address.startsWith("C")) { throw new Error("unsupported"); }
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"named-alias", ".js", `import { StrKey as Keys } from "@stellar/stellar-sdk";
function decode(address) {
  if (address[0] !== "C") return null;
  return Keys.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"namespace-import", ".js", `import * as Stellar from "@stellar/stellar-sdk";
function valid(address) {
  if (!address.startsWith("G")) return false;
  return Stellar.StrKey.isValidMuxedContract(address);
}`, "isValidMuxedContract"},
		{"base-subpath", ".js", `import { StrKey } from "@stellar/stellar-sdk/base";
function decode(address) {
  if (!address.startsWith("M")) return undefined;
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"typescript-function", ".ts", `import { StrKey } from "@stellar/stellar-sdk";
export function decode(address: string): Uint8Array {
  if (!/^[GMC][A-Z2-7]+$/.test(address)) throw new Error("unsupported");
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"typescript-arrow", ".ts", `import { StrKey } from "@stellar/stellar-sdk";
export const decode = (address: string) => {
  if (!/^(G|M|C)[A-Z2-7]+$/.test(address)) { throw new Error("unsupported"); }
  return StrKey.decodeMuxedContract(address);
};`, "decodeMuxedContract"},
		{"function-expression", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const valid = function(address) {
  if (address[0] === "W") return false;
  return StrKey.isValidMuxedContract(address);
};`, "isValidMuxedContract"},
		{"prefix-disjunction", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) {
  if (!(address.startsWith("G") || address.startsWith("M") || address.startsWith("C"))) throw new Error("unsupported");
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"prefix-conjunction", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) {
  if (!address.startsWith("G") && !address.startsWith("M") && !address.startsWith("C")) throw new Error("unsupported");
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"jsx-neighbor", ".jsx", `import { StrKey } from "@stellar/stellar-sdk";
const View = () => <span>W address support</span>;
function decode(address) {
  if (!address.startsWith("C")) throw new Error("unsupported");
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"tsx-neighbor", ".tsx", `import { StrKey } from "@stellar/stellar-sdk";
const View = () => <span>W address support</span>;
function decode(address: string) {
  if (!address.startsWith("C")) throw new Error("unsupported");
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
		{"comments-are-not-statements", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) {
  // Reject old prefixes only: this comment itself is not executable evidence.
  if (!address.startsWith("C")) { /* deliberate guard */ throw new Error("unsupported"); }
  // The decoder consumes the same parameter.
  return StrKey.decodeMuxedContract(address);
}`, "decodeMuxedContract"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			file := jsAddressInput(tc.source, tc.extension)
			got, _, err := analyzeJSAddress(file)
			if err != nil || len(got) != 1 {
				t.Fatalf("expected one proven local candidate, got %+v, %v", got, err)
			}
			candidate := got[0]
			if candidate.path != file.Path || candidate.line <= 0 || candidate.evidence == "" || candidate.parameter != "address" || candidate.ecosystem != "npm" || candidate.kind != addressWOnly || !strings.Contains(candidate.api, tc.sink) {
				t.Fatalf("incorrect local evidence: %+v", candidate)
			}
		})
	}
}

func TestJSAddressSACDestinationCandidates(t *testing.T) {
	for _, source := range []string{
		`import { TransactionBuilder } from "@stellar/stellar-sdk";
function transfer(address, source, asset) {
  const builder = new TransactionBuilder(source, {fee: "100", networkPassphrase: "example"});
  if (!address.startsWith("G")) throw new Error("accounts only");
  return builder.addSacTransferOperation(address, asset, "1");
}`,
		`import { TransactionBuilder as Builder } from "@stellar/stellar-sdk";
function transfer(address, source, asset) {
  if (!address.startsWith("G")) throw new Error("accounts only");
  return new Builder(source, {fee: "100", networkPassphrase: "example"}).addSacTransferOperation(address, asset, "1");
}`,
		`import * as Stellar from "@stellar/stellar-sdk";
function transfer(address, source, asset) {
  const builder = new Stellar.TransactionBuilder(source, {fee: "100", networkPassphrase: "example"});
  if (address[0] !== "G") return null;
  return builder.addSacTransferOperation(address, asset, "1");
}`,
	} {
		got, _, err := analyzeJSAddress(jsAddressInput(source, ".js"))
		if err != nil || len(got) != 1 || got[0].kind != addressSACDestination || got[0].parameter != "address" || !strings.Contains(got[0].api, "addSacTransferOperation") {
			t.Fatalf("expected lower-context SAC candidate, got %+v, %v", got, err)
		}
	}
}

func TestJSAddressNegativeContexts(t *testing.T) {
	tests := []struct{ name, extension, source string }{
		{"modern-prefix", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("W")) throw new Error("not W"); return StrKey.decodeMuxedContract(address); }`},
		{"escaped-w-unicode-prefix", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("\u0057")) throw new Error("not W"); return StrKey.decodeMuxedContract(address); }`},
		{"escaped-w-hex-prefix", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith('\x57')) throw new Error("not W"); return StrKey.decodeMuxedContract(address); }`},
		{"escaped-w-regex", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^\x57[A-Z2-7]+$/.test(address)) throw new Error("not W"); return StrKey.decodeMuxedContract(address); }`},
		{"modern-regex", ".ts", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address: string) { if (!/^[WGMC][A-Z2-7]+$/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"different-parameter", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address, account) { if (!account.startsWith("G")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"different-sink-value", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address, other) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(other); }`},
		{"prefix-alone", ".js", `function validateAccount(address) { if (!address.startsWith("G")) throw new Error("bad"); return address; }`},
		{"c-only-decoder", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeContract(address); }`},
		{"account-only-payment", ".js", `import { Operation } from "@stellar/stellar-sdk";
function pay(address, asset) { if (!/^[GM]/.test(address)) throw new Error("bad"); return Operation.payment({destination: address, asset, amount: "1"}); }`},
		{"address-object-context", ".js", `import { Address } from "@stellar/stellar-sdk";
function storageKey(address) { if (!address.startsWith("C")) throw new Error("bad"); return Address.fromString(address); }`},
		{"generic-native-address", ".js", `import { nativeToScVal } from "@stellar/stellar-sdk";
function key(address) { if (!address.startsWith("C")) throw new Error("bad"); return nativeToScVal(address, {type: "address"}); }`},
		{"foreign-sdk", ".js", `import { StrKey } from "my-stellar-fork";
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"no-sdk-import", ".js", `function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"shadowed-parameter", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address, StrKey) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"shadowed-namespace", ".js", `import * as Stellar from "@stellar/stellar-sdk";
function decode(address, Stellar) { if (!address.startsWith("C")) throw new Error("bad"); return Stellar.StrKey.decodeMuxedContract(address); }`},
		{"local-sdk-binding", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { const StrKey = customKeys; if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"type-only-import", ".ts", `import type { StrKey } from "@stellar/stellar-sdk";
function decode(address: string) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"type-only-specifier", ".ts", `import { type StrKey } from "@stellar/stellar-sdk";
function decode(address: string) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"reassigned-value", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); address = "W"; return StrKey.decodeMuxedContract(address); }`},
		{"overwritten-sdk-method", ".js", `import { StrKey } from "@stellar/stellar-sdk";
StrKey.decodeMuxedContract = address => address;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"overwritten-sdk-alias-member", ".js", `import { StrKey as Keys } from "@stellar/stellar-sdk";
Keys["decodeMuxedContract"] = address => address;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return Keys.decodeMuxedContract(address); }`},
		{"namespace-member-mutation", ".js", `import * as Stellar from "@stellar/stellar-sdk";
Stellar.StrKey.decodeMuxedContract = address => address;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return Stellar.StrKey.decodeMuxedContract(address); }`},
		{"builtin-prefix-method-mutation", ".js", `import { StrKey } from "@stellar/stellar-sdk";
String.prototype.startsWith = function() { return true; };
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"else-bypass", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C")) { throw new Error("bad"); } else { return null; } return StrKey.decodeMuxedContract(address); }`},
		{"caught-guard", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { try { if (!address.startsWith("C")) throw new Error("bad"); } catch (_) {} return StrKey.decodeMuxedContract(address); }`},
		{"nested-closure", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function outer() { function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); } return decode; }`},
		{"intervening-call", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); log(address); return StrKey.decodeMuxedContract(address); }`},
		{"empty-rejection", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C")) {} return StrKey.decodeMuxedContract(address); }`},
		{"arbitrary-rejection-return", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C")) return fallback(address); return StrKey.decodeMuxedContract(address); }`},
		{"side-effect-condition", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C") && callback()) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"short-circuit-side-effect-condition", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith("C") || callback()) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"assignment-condition", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if ((address = other) && !address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"unanchored-regex", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/[GMC]/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"unsupported-lookahead", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^(?!W)[A-Z2-7]+$/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"unsupported-regex-flags", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^[GMC]/g.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"positive-c-branch-does-not-reject-w", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"static-sac-method", ".js", `import { TransactionBuilder } from "@stellar/stellar-sdk";
function transfer(address, asset) { if (!address.startsWith("G")) throw new Error("accounts only"); return TransactionBuilder.addSacTransferOperation(address, asset, "1"); }`},
		{"sac-object-instead-of-first-string-argument", ".js", `import { TransactionBuilder } from "@stellar/stellar-sdk";
function transfer(address, source, asset) { const builder = new TransactionBuilder(source, {}); if (!address.startsWith("G")) throw new Error("accounts only"); return builder.addSacTransferOperation({destination: address, asset, amount: "1"}); }`},
		{"unknown-builder-instance", ".js", `function transfer(address, builder, asset) { if (!address.startsWith("G")) throw new Error("accounts only"); return builder.addSacTransferOperation(address, asset, "1"); }`},
		{"comment-fake-code", ".js", `import { StrKey } from "@stellar/stellar-sdk";
// function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }
function good(address) { return StrKey.decodeMuxedContract(address); }`},
		{"string-fake-code", ".js", `const example = "import { StrKey } from '@stellar/stellar-sdk'; function decode(address) { if (!address.startsWith('C')) throw new Error('bad'); return StrKey.decodeMuxedContract(address); }";`},
		{"template-fake-code", ".js", "const example = `import { StrKey } from '@stellar/stellar-sdk'; function decode(address) { if (!address.startsWith('C')) throw new Error('bad'); return StrKey.decodeMuxedContract(address); }`;"},
		{"side-effecting-parameter-default", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address = callback()) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"shadowed-undefined-parameter", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address, undefined) { if (!address.startsWith("C")) return undefined; return StrKey.decodeMuxedContract(address); }`},
		{"shadowed-undefined-module-binding", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const undefined = fallbackValue;
function decode(address) { if (!address.startsWith("C")) return undefined; return StrKey.decodeMuxedContract(address); }`},
		{"shadowed-undefined-typed-array", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const undefined = new Uint8Array(72);
function decode(address) { if (!address.startsWith("C")) return undefined; return StrKey.decodeMuxedContract(address); }`},
		{"shadowed-undefined-named-expression", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const decode = function undefined(address) { if (!address.startsWith("C")) return undefined; return StrKey.decodeMuxedContract(address); }`},
		{"globalthis-prototype-write", ".js", `import { StrKey } from "@stellar/stellar-sdk";
globalThis.String.prototype.startsWith = () => true;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"global-object-alias", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const globals = globalThis; globals.String.prototype.startsWith = () => true;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"global-object-container-escape", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const box = {globals: globalThis}; box.globals.String.prototype.startsWith = () => true;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"global-function-constructor", ".js", `import { StrKey } from "@stellar/stellar-sdk";
new globalThis.Function("String.prototype.startsWith = () => true")();
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"aliased-function-constructor", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const Construct = Function; new Construct("String.prototype.startsWith = () => true")();
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"cast-sdk-mutation", ".ts", `import { StrKey } from "@stellar/stellar-sdk";
Object.assign(StrKey as any, {decodeMuxedContract: (x: string) => x});
function decode(address: string) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"sdk-container-escape", ".js", `import { StrKey } from "@stellar/stellar-sdk";
patch({sdk: StrKey});
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"sdk-shorthand-escape", ".js", `import { StrKey } from "@stellar/stellar-sdk";
const tools = {StrKey}; tools.StrKey.decodeMuxedContract = x => x;
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"go-only-begin-escape", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/\AC/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"go-only-quote-escape", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^\QC\E/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"go-posix-class", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^[^[:alpha:]]?A/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"go-unicode-property-class", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^[^\p{Lu}]?A/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"go-brace-hex-class", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!/^[^\x{57}]?A/.test(address)) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"optional-guard-call", ".js", `import { StrKey } from "@stellar/stellar-sdk";
function decode(address) { if (!address.startsWith?.("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"opaque-eval-mutation", ".js", `import { StrKey } from "@stellar/stellar-sdk";
eval("StrKey.decodeMuxedContract = x => x");
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
		{"opaque-function-constructor", ".js", `import { StrKey } from "@stellar/stellar-sdk";
new Function("String.prototype.startsWith = () => true")();
function decode(address) { if (!address.startsWith("C")) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := analyzeJSAddress(jsAddressInput(tc.source, tc.extension))
			if err != nil || len(got) != 0 {
				t.Fatalf("unproven or valid context produced candidates: %+v, %v", got, err)
			}
		})
	}
}

func TestJSAddressEvidenceLocationsAndOrdering(t *testing.T) {
	source := `import { StrKey } from "@stellar/stellar-sdk";
function first(address) {
  if (!address.startsWith("C")) throw new Error("bad");
  return StrKey.decodeMuxedContract(address);
}
function second(address) {
  if (!address.startsWith("G")) throw new Error("bad");
  return StrKey.decodeMuxedContract(address);
}`
	file := jsAddressInput(source, ".js")
	got, _, err := analyzeJSAddress(file)
	if err != nil || len(got) != 2 {
		t.Fatalf("expected two independent candidates, got %+v, %v", got, err)
	}
	if got[0].line != 3 || got[1].line != 7 || path.Base(got[0].path) != "sample.js" || !strings.Contains(got[0].evidence, "startsWith") {
		t.Fatalf("incorrect source positions/evidence: %+v", got)
	}
}

func TestJSAddressInvalidSyntaxIsExplicitLimitation(t *testing.T) {
	for _, extension := range []string{".js", ".jsx", ".ts", ".tsx"} {
		t.Run(extension, func(t *testing.T) {
			got, limitations, err := analyzeJSAddress(jsAddressInput(`import { StrKey } from "@stellar/stellar-sdk"; function broken( {`, extension))
			if err != nil || len(got) != 0 || len(limitations) == 0 {
				t.Fatalf("invalid source must remain unclassified with a limitation: %+v, %v, %v", got, limitations, err)
			}
		})
	}
}

func TestJSAddressAdditionalLiteralPrefixForms(t *testing.T) {
	for _, condition := range []string{`address.charAt(0) !== "C"`, `!["G", "M", "C"].includes(address[0])`, `!["G", "M", "C"].includes(address.charAt(0))`} {
		source := `import { StrKey } from "@stellar/stellar-sdk"; function decode(address) { if (` + condition + `) throw new Error("bad"); return StrKey.decodeMuxedContract(address); }`
		got, _, err := analyzeJSAddress(jsAddressInput(source, ".js"))
		if err != nil || len(got) != 1 {
			t.Fatalf("supported prefix form %q: %+v %v", condition, got, err)
		}
	}
}

func TestJSAddressNodeBudgetIsAnError(t *testing.T) {
	_, notes, err := analyzeJSAddress(jsAddressInput(strings.Repeat("0;", 70000), ".js"))
	if err == nil || len(notes) != 0 || (!strings.Contains(err.Error(), "nodes") && !strings.Contains(err.Error(), "two-second budget")) {
		t.Fatalf("resource exhaustion was not explicit: %v %v", notes, err)
	}
}
