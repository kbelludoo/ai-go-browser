'use strict';

const readline = require('node:readline');
const { performance } = require('node:perf_hooks');
const lines = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
let browser, page, config, chromiumPID = 0;

function send(value) { process.stdout.write(JSON.stringify(value) + '\n'); }

async function initialize(message) {
  config = message;
  const playwright = require(config.module);
  browser = await playwright.chromium.launch({ executablePath: config.browser, headless: true });
  const context = await browser.newContext({ viewport: { width: 1366, height: 768 } });
  page = await context.newPage();
  let pidSource = 'unavailable';
  try {
    const session = await browser.newBrowserCDPSession();
    const result = await session.send('SystemInfo.getProcessInfo');
    const main = result.processInfo.find(p => p.type.toLowerCase() === 'browser');
    if (main && Number.isInteger(main.id) && main.id > 0) {
      chromiumPID = main.id;
      pidSource = 'CDP SystemInfo.getProcessInfo';
    }
    await session.detach();
  } catch (error) { pidSource = String(error.message || error); }
  send({ ok: true, node_pid: process.pid, chromium_pid: chromiumPID, chromium_pid_source: pidSource,
    node_version: process.version, chromium_version: browser.version() });
}

async function task(fixture) {
  const start = performance.now();
  const deadline = start + fixture.timeout_ms;
  const remaining = () => Math.max(1, Math.floor(deadline - performance.now()));
  let snapshot, error = '';
  try {
    await page.goto(fixture.url, { waitUntil: 'load', timeout: remaining() });
    switch (fixture.action) {
      case 'form':
        await page.locator('#q').fill(fixture.value, { timeout: remaining() });
        await page.locator('#search button').click({ timeout: remaining() });
        break;
      case 'link':
        await page.locator('#continue').click({ timeout: remaining() });
        break;
      case 'spa':
        await page.locator('#name').fill(fixture.value, { timeout: remaining() });
        await page.locator('#go').click({ timeout: remaining() });
        break;
      case '': break;
      default: throw new Error('unknown fixture action ' + fixture.action);
    }
    await page.waitForFunction(expected => (document.body?.innerText || '').includes(expected),
      fixture.expected, { polling: 50, timeout: remaining() });
    snapshot = await page.evaluate(config.snapshot_script);
    if (!snapshot.text.includes(fixture.expected)) throw new Error('serialized compact snapshot did not contain expected visible text');
  } catch (failure) { error = String(failure.message || failure); }
  const taskMS = performance.now() - start;
  const memory = process.memoryUsage();
  send({ ok: error === '', error, task_ms: taskMS, snapshot,
    snapshot_bytes: snapshot ? Buffer.byteLength(JSON.stringify(snapshot), 'utf8') : 0,
    node_heap_used_bytes: memory.heapUsed, node_rss_bytes: memory.rss,
    node_external_bytes: memory.external, node_array_buffer_bytes: memory.arrayBuffers });
}

(async () => {
  try {
    for await (const line of lines) {
      if (!line.trim()) continue;
      const message = JSON.parse(line);
      if (message.cmd === 'init') await initialize(message);
      else if (message.cmd === 'task') await task(message);
      else if (message.cmd === 'close') {
        if (browser) await browser.close();
        send({ ok: true });
        lines.close();
        return;
      } else throw new Error('unknown worker command');
    }
  } catch (error) {
    send({ ok: false, error: String(error.message || error) });
    process.exitCode = 1;
  } finally {
    if (browser) await browser.close().catch(() => {});
  }
})();
