'use strict';
const readline = require('node:readline');
const { performance } = require('node:perf_hooks');
const lines = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
let browser, page, config;
const send = value => process.stdout.write(JSON.stringify(value) + '\n');

async function initialize(message) {
  config = message;
  const playwright = require(config.module);
  browser = await playwright.chromium.launch({ executablePath: config.browser, headless: true });
  page = await (await browser.newContext({ viewport: { width: 1366, height: 768 } })).newPage();
  let chromiumPID = 0, pidSource = 'unavailable';
  try {
    const session = await browser.newBrowserCDPSession();
    const result = await session.send('SystemInfo.getProcessInfo');
    const main = result.processInfo.find(p => p.type.toLowerCase() === 'browser');
    if (main && Number.isInteger(main.id) && main.id > 0) { chromiumPID = main.id; pidSource = 'CDP SystemInfo.getProcessInfo'; }
    await session.detach();
  } catch (error) { pidSource = String(error.message || error); }
  send({ ok: true, chromium_pid: chromiumPID, chromium_pid_source: pidSource, chromium_version: browser.version() });
}

async function task(fixture) {
  const start = performance.now(), deadline = start + fixture.timeout_ms;
  const remaining = () => Math.max(1, Math.floor(deadline - performance.now()));
  let snapshot, error = '';
  try {
    await page.goto(fixture.url, { waitUntil: 'load', timeout: remaining() });
    let scope = page, locators = page;
    if (fixture.frame_name) {
      locators = page.frameLocator('iframe[name="' + fixture.frame_name + '"]');
      await locators.locator('#frame-note').waitFor({ state: 'visible', timeout: remaining() });
      scope = page.frame({ name: fixture.frame_name });
      if (!scope) throw new Error('named frame not available: ' + fixture.frame_name);
    }
    const assertText = async expected => {
      await scope.waitForFunction(expected => (document.body?.innerText || '').includes(expected), expected,
        { polling: 50, timeout: remaining() });
    };
    for (const step of fixture.steps) {
      if (step.kind === 'fill') await locators.locator(step.selector).fill(step.value, { timeout: remaining() });
      else if (step.kind === 'press') await locators.locator(step.selector).click({ timeout: remaining() });
      else if (step.kind === 'assert') await assertText(step.expected);
      else throw new Error('unknown modern task step ' + step.kind);
    }
    await assertText(fixture.expected);
    snapshot = await scope.evaluate(config.snapshot_script);
    if (!snapshot.text.includes(fixture.expected)) throw new Error('compact snapshot omitted the task result');
  } catch (failure) { error = String(failure.message || failure); }
  const memory = process.memoryUsage();
  send({ ok: error === '', error, task_ms: performance.now() - start, snapshot,
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
      else if (message.cmd === 'close') { await browser?.close(); send({ ok: true }); lines.close(); return; }
      else throw new Error('unknown modern worker command');
    }
  } catch (error) { send({ ok: false, error: String(error.message || error) }); process.exitCode = 1; }
  finally { await browser?.close().catch(() => {}); }
})();
