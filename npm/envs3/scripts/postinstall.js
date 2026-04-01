const os = require("os");
const fs = require("fs");
const path = require("path");
const https = require("https");

// Platform package mapping (Node naming)
const PLATFORM_MAP = {
  "darwin-arm64": "@mucan54/envs3-darwin-arm64",
  "darwin-x64": "@mucan54/envs3-darwin-x64",
  "linux-x64": "@mucan54/envs3-linux-x64",
  "linux-arm64": "@mucan54/envs3-linux-arm64",
  "win32-x64": "@mucan54/envs3-win32-x64",
};

// Go target naming (for GitHub Release assets)
const GO_TARGET_MAP = {
  "darwin-arm64": "envs3-darwin-arm64",
  "darwin-x64": "envs3-darwin-amd64",
  "linux-x64": "envs3-linux-amd64",
  "linux-arm64": "envs3-linux-arm64",
  "win32-x64": "envs3-windows-amd64.exe",
};

const key = `${os.platform()}-${os.arch()}`;
const pkg = PLATFORM_MAP[key];

// Check if platform binary already installed via optionalDependencies
if (pkg) {
  try {
    require.resolve(`${pkg}/package.json`);
    process.exit(0); // Already installed, nothing to do
  } catch {}
}

// Platform not available via optionalDependencies — download from GitHub Releases
const target = GO_TARGET_MAP[key];
if (!target) {
  console.warn(`envs3: unsupported platform ${key}, skipping binary download`);
  process.exit(0);
}

const version = require("../package.json").version;
if (version === "0.0.0") {
  console.warn("envs3: development version, skipping binary download");
  process.exit(0);
}

const url = `https://github.com/mucan54/envs3/releases/download/v${version}/${target}`;
const binName = os.platform() === "win32" ? "envs3-native.exe" : "envs3-native";
const dest = path.join(__dirname, "..", "bin", binName);

console.log(`envs3: downloading binary for ${key}...`);

function download(url, redirects) {
  if (redirects > 5) {
    console.warn("envs3: too many redirects, skipping download");
    process.exit(0);
  }

  https.get(url, { headers: { "User-Agent": "envs3-npm" } }, (res) => {
    // Follow redirects (GitHub sends 302 to CDN)
    if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
      download(res.headers.location, redirects + 1);
      return;
    }

    if (res.statusCode !== 200) {
      console.warn(`envs3: download failed (HTTP ${res.statusCode}), skipping`);
      process.exit(0);
    }

    const chunks = [];
    res.on("data", (chunk) => chunks.push(chunk));
    res.on("end", () => {
      const data = Buffer.concat(chunks);

      // Ensure directory exists
      fs.mkdirSync(path.dirname(dest), { recursive: true });

      // Write binary
      fs.writeFileSync(dest, data, { mode: 0o755 });

      console.log(`envs3: binary installed (${(data.length / 1024 / 1024).toFixed(1)} MB)`);
    });
    res.on("error", (err) => {
      console.warn(`envs3: download error: ${err.message}, skipping`);
      process.exit(0);
    });
  }).on("error", (err) => {
    console.warn(`envs3: download error: ${err.message}, skipping`);
    process.exit(0);
  });
}

download(url, 0);
