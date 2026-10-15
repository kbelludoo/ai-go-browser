import express from 'express';
import { createServer as createViteServer } from 'vite';
import path from 'path';

async function startServer() {
  const app = express();
  const PORT = 3000;

  app.use(express.json());

  // Proxy endpoint to fetch real websites securely without client CORS restrictions
  app.get('/api/proxy', async (req, res) => {
    const targetUrl = req.query.url as string;
    if (!targetUrl) {
      return res.status(400).json({ error: 'Missing url parameter' });
    }

    try {
      const parsedUrl = new URL(targetUrl);
      if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
        return res.status(400).json({ error: 'Only http and https protocols are supported' });
      }

      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 12000);

      const response = await fetch(targetUrl, {
        signal: controller.signal,
        headers: {
          'User-Agent':
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 AI-Go-Browser/0.7',
          Accept:
            'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8',
          'Accept-Language': 'en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7',
        },
        redirect: 'follow',
      });
      clearTimeout(timeout);

      const contentType = response.headers.get('content-type') || 'text/html';
      const htmlText = await response.text();

      return res.json({
        url: response.url || targetUrl,
        status: response.status,
        contentType,
        bytes: Buffer.byteLength(htmlText, 'utf8'),
        html: htmlText,
      });
    } catch (err: any) {
      const errorMsg = err.name === 'AbortError' ? 'Request timeout after 12s' : err.message;
      return res.status(502).json({
        error: `Failed to fetch target URL: ${errorMsg}`,
        url: targetUrl,
      });
    }
  });

  // Batch webtest endpoint to measure real sites
  app.post('/api/webtest', async (req, res) => {
    const urls: string[] = req.body.urls || [
      'https://example.com/',
      'https://news.ycombinator.com/',
      'https://go.dev/',
      'https://en.wikipedia.org/wiki/Go_(programming_language)',
      'https://httpbin.org/',
      'https://books.toscrape.com/',
      'https://lite.cnn.com/',
      'https://pkg.go.dev/',
    ];

    const results = [];

    for (const url of urls) {
      const startTime = Date.now();
      try {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), 10000);

        const resp = await fetch(url, {
          signal: controller.signal,
          headers: {
            'User-Agent':
              'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0.0.0 Safari/537.36 AI-Go-Browser/0.7',
            Accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
          },
          redirect: 'follow',
        });
        clearTimeout(timeout);

        const text = await resp.text();
        const duration = Date.now() - startTime;
        const bytes = Buffer.byteLength(text, 'utf8');

        results.push({
          url,
          loaded: resp.status >= 200 && resp.status < 400,
          status: resp.status,
          bytes,
          duration_ms: duration,
          html_snippet: text.substring(0, 1000),
          html: text,
        });
      } catch (err: any) {
        results.push({
          url,
          loaded: false,
          status: 0,
          error: err.name === 'AbortError' ? 'Timeout' : err.message,
          duration_ms: Date.now() - startTime,
          bytes: 0,
          html: '',
        });
      }
    }

    return res.json({ results });
  });

  // Mount Vite middleware in development
  const isProduction = process.env.NODE_ENV === 'production';
  if (!isProduction) {
    const vite = await createViteServer({
      server: { middlewareMode: true },
      appType: 'spa',
    });
    app.use(vite.middlewares);
  } else {
    app.use(express.static(path.resolve(__dirname, 'dist')));
    app.get('*', (_req, res) => {
      res.sendFile(path.resolve(__dirname, 'dist', 'index.html'));
    });
  }

  app.listen(PORT, '0.0.0.0', () => {
    console.log(`Server running at http://0.0.0.0:${PORT}`);
  });
}

startServer();
