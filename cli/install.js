#!/usr/bin/env node
// Downloads the correct thask binary from GitHub Releases at install time.

const { execSync } = require('child_process');
const https = require('https');
const fs = require('fs');
const path = require('path');

const VERSION = require('./package.json').version;
const REPO = 'kimgh06/Thask';

const PLATFORM_MAP = {
  'darwin-arm64':  'thask-darwin-arm64',
  'darwin-x64':    'thask-darwin-amd64',
  'linux-arm64':   'thask-linux-arm64',
  'linux-x64':     'thask-linux-amd64',
  'win32-x64':     'thask-windows-amd64.exe',
};

const key = `${process.platform}-${process.arch}`;
const binaryName = PLATFORM_MAP[key];

if (!binaryName) {
  console.error(`[thask] Unsupported platform: ${key}`);
  process.exit(0); // non-fatal — user can install manually
}

const nativeDir = path.join(__dirname, 'native');
const ext = process.platform === 'win32' ? '.exe' : '';
const dest = path.join(nativeDir, `thask${ext}`);

if (fs.existsSync(dest)) {
  process.exit(0); // already installed
}

fs.mkdirSync(nativeDir, { recursive: true });

const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;
console.log(`[thask] Downloading ${binaryName} from GitHub Releases...`);

function download(url, dest, cb) {
  const file = fs.createWriteStream(dest);
  https.get(url, (res) => {
    if (res.statusCode === 302 || res.statusCode === 301) {
      file.close();
      fs.unlinkSync(dest);
      download(res.headers.location, dest, cb);
      return;
    }
    if (res.statusCode !== 200) {
      file.close();
      fs.unlinkSync(dest);
      cb(new Error(`HTTP ${res.statusCode}`));
      return;
    }
    res.pipe(file);
    file.on('finish', () => file.close(cb));
  }).on('error', (err) => {
    fs.unlinkSync(dest);
    cb(err);
  });
}

download(url, dest, (err) => {
  if (err) {
    console.error(`[thask] Download failed: ${err.message}`);
    console.error(`[thask] Install manually: https://github.com/${REPO}/releases/tag/v${VERSION}`);
    process.exit(0); // non-fatal
  }
  fs.chmodSync(dest, 0o755);
  console.log(`[thask] Installed successfully.`);
});
