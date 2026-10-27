const GRADES = ["A", "B", "C"];
const SECTION_RE = /^[A-C]\d+$/;
const half = total / 2 / count;
const css = `color: ${GRADES.length > 1 ? "red" : "blue"}`;

export function grade(score) {
  if (score.startsWith("B") || score.startsWith("C")) {
    return GRADES.indexOf(score) + half;
  }
  return SECTION_RE.test(score) ? css : null;
}
