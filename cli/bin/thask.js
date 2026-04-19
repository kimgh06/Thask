#!/usr/bin/env node
const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const ext = process.platform === 'win32' ? '.exe' : '';
const binary = path.join(__dirname, '..', 'native', `thask${ext}`);

if (!fs.existsSync(binary)) {
  console.error('[thask] Binary not found. Try reinstalling: npm install -g thask');
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit' });
process.exit(result.status ?? 1);
