const fs = require("node:fs/promises");
const path = require("node:path");
const sharp = require("sharp");

async function main() {
  const root = path.resolve(__dirname, "..");
  const source = path.join(root, "public", "metronic", "upload", "aeplogonew.png");
  const target = path.join(root, "app", "favicon.ico");
  const size = 64;
  const { data: rgba } = await sharp(source)
    .extract({ left: 0, top: 0, width: 1299, height: 1250 })
    .resize(size, size, { fit: "contain", background: "#ffffff" })
    .ensureAlpha()
    .raw()
    .toBuffer({ resolveWithObject: true });

  const dibHeader = Buffer.alloc(40);
  dibHeader.writeUInt32LE(40, 0);
  dibHeader.writeInt32LE(size, 4);
  dibHeader.writeInt32LE(size * 2, 8);
  dibHeader.writeUInt16LE(1, 12);
  dibHeader.writeUInt16LE(32, 14);
  const pixels = Buffer.alloc(size * size * 4);
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const src = (y * size + x) * 4;
      const dst = ((size - 1 - y) * size + x) * 4;
      pixels[dst] = rgba[src + 2];
      pixels[dst + 1] = rgba[src + 1];
      pixels[dst + 2] = rgba[src];
      pixels[dst + 3] = rgba[src + 3];
    }
  }
  const mask = Buffer.alloc((size / 8) * size);
  dibHeader.writeUInt32LE(pixels.length + mask.length, 20);
  const bitmap = Buffer.concat([dibHeader, pixels, mask]);

  const header = Buffer.alloc(22);
  header.writeUInt16LE(1, 2);
  header.writeUInt16LE(1, 4);
  header.writeUInt8(size, 6);
  header.writeUInt8(size, 7);
  header.writeUInt16LE(1, 10);
  header.writeUInt16LE(32, 12);
  header.writeUInt32LE(bitmap.length, 14);
  header.writeUInt32LE(header.length, 18);
  await fs.writeFile(target, Buffer.concat([header, bitmap]));
  process.stdout.write(`Wrote ${target}\n`);
}

main().catch((error) => {
  process.stderr.write(`${error}\n`);
  process.exitCode = 1;
});
