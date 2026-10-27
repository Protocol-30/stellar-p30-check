// Tests may hold narrow validators on purpose; they are not scanned.
const OLD = /^[GMC][A-Z2-7]{55}$/;
test("old", () => expect(OLD.test("G")).toBe(false));
