const net = require('net');
const path = require('path');
const { spawn } = require('child_process');

const host = process.env.CSF_FRONTEND_HOST || '127.0.0.1';
const port = Number.parseInt(process.env.CSF_FRONTEND_DEV_PORT || '10001', 10);
const frontendRoot = path.resolve(__dirname, '..');

function ensurePortIsFree(bindHost, bindPort) {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once('error', (error) => {
      if (error && error.code === 'EADDRINUSE') {
        reject(new Error(`Frontend dev port ${bindHost}:${bindPort} is already in use.`));
        return;
      }
      reject(error);
    });
    server.once('listening', () => {
      server.close((closeError) => {
        if (closeError) {
          reject(closeError);
          return;
        }
        resolve();
      });
    });
    server.listen(bindPort, bindHost);
  });
}

async function main() {
  await ensurePortIsFree(host, port);

  const child = spawn(
    process.platform === 'win32' ? 'npx.cmd' : 'npx',
    ['quasar', 'dev', '--hostname', host, '--port', String(port)],
    {
      cwd: frontendRoot,
      env: process.env,
      stdio: 'inherit',
    }
  );

  child.on('exit', (code) => {
    process.exit(code ?? 0);
  });

  child.on('error', (error) => {
    console.error(error.message);
    process.exit(1);
  });
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
