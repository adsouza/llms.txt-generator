export function generateLlmsTxtStream(url, callbacks) {
  const { onDiscovered, onProgress, onDone, onError } = callbacks;

  const controller = new AbortController();

  (async () => {
    try {
      const resp = await fetch('/api/generate-stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
        signal: controller.signal,
      });

      if (!resp.ok) {
        const text = await resp.text();
        onError(text || 'Generation failed');
        return;
      }

      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });

        const parts = buffer.split('\n\n');
        buffer = parts.pop();

        for (const part of parts) {
          const lines = part.split('\n');
          let eventType = '';
          let data = '';

          for (const line of lines) {
            if (line.startsWith('event: ')) {
              eventType = line.slice(7);
            } else if (line.startsWith('data: ')) {
              data = line.slice(6);
            }
          }

          if (!eventType || !data) continue;

          const parsed = JSON.parse(data);

          switch (eventType) {
            case 'discovered':
              onDiscovered(parsed.URLs, parsed.Total);
              break;
            case 'progress':
              onProgress(parsed.CurrentURL, parsed.Done, parsed.Total);
              break;
            case 'done':
              onDone(parsed.Result);
              break;
            case 'error':
              onError(parsed.Error);
              break;
          }
        }
      }
    } catch (err) {
      if (err.name !== 'AbortError') {
        onError(err.message);
      }
    }
  })();

  return () => controller.abort();
}
