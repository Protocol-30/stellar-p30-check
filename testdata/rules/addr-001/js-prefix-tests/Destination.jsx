export function Destination({ dest }) {
  if (!(dest.startsWith("G") || dest.startsWith("M") || dest.startsWith("C"))) {
    return <p>Don't send here: unsupported address</p>;
  }
  return <p>Sending to {dest}</p>;
}
