import * as crypto from "crypto";

const text = "123abc";
const start = Date.now();

let iteration = 0;

while (true) {
  const candidate = `${text} ${iteration}`;
  const sha256Hash = crypto
    .createHash("sha256")
    .update(candidate)
    .digest("hex");

  if (sha256Hash.startsWith("000000")) {
    console.log(`\nFound: ${candidate}`);
    console.log(`SHA256: ${sha256Hash}`);
    break;
  }
  iteration++;
}

const elapsed = Date.now() - start;
console.log(`\nTime taken: ${elapsed}ms`);
