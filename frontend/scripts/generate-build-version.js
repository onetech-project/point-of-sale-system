const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const appRoot = path.resolve(__dirname, '..');
const publicDir = path.join(appRoot, 'public');
const packageJsonPath = path.join(appRoot, 'package.json');
const versionJsonPath = path.join(publicDir, 'version.json');

const readPackageVersion = () => {
  try {
    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
    return packageJson.version || '0.0.0';
  } catch {
    return '0.0.0';
  }
};

const readGitCommit = () => {
  try {
    return execFileSync('git', ['rev-parse', '--short=12', 'HEAD'], {
      cwd: appRoot,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim();
  } catch {
    return null;
  }
};

const buildTime = new Date().toISOString();
const appVersion = readPackageVersion();
const commit = readGitCommit();
const hashInput = [appVersion, commit || 'unknown', buildTime].join(':');
const buildHash = crypto.createHash('sha256').update(hashInput).digest('hex').slice(0, 16);

const buildVersion = {
  buildHash,
  buildTime,
  appVersion,
  commit,
};

fs.mkdirSync(publicDir, { recursive: true });
fs.writeFileSync(versionJsonPath, `${JSON.stringify(buildVersion, null, 2)}\n`);

console.log(`Generated public/version.json for build ${buildHash}`);
