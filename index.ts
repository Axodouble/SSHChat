import * as crypto from "crypto";

const text = "Hello, World!";
const start = Date.now();

let iteration = 0;

while (true) {
  const candidate = `${text} (ITERATION: ${iteration})`;
  const sha256Hash = crypto
    .createHash("sha256")
    .update(candidate)
    .digest("hex");
  if (sha256Hash.startsWith("00000")) {
    console.log(`\nFound: ${candidate}`);
    console.log(`SHA256: ${sha256Hash}`);
    break;
  }
  iteration++;
}

const elapsed = Date.now() - start;
console.log(`\nTime taken: ${elapsed}ms`);
