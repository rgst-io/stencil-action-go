'use strict';
const os = require('os');
const childProcess = require('child_process');

const archMap = {
	'x64': 'amd64',
	'arm64': 'arm64',
	'arm': 'arm',
	'ia32': '386'
};

const platform = os.platform();
const arch = os.arch();

const GOOS = platform === 'win32' ? 'windows' : platform;
const GOARCH = archMap[arch] || arch;

const binary = `${__dirname}/main-${GOOS}-${GOARCH}${GOOS === 'windows' ? '.exe' : ''}`
childProcess.spawnSync(binary, { stdio: 'inherit' })
