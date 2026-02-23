<script>
  import { generateLlmsTxtStream } from './lib/api.js';

  let state = $state('idle');
  let result = $state('');
  let errorMsg = $state('');
  let url = $state('');
  let copied = $state(false);
  let discoveredURLs = $state([]);
  let crawlDone = $state(0);
  let crawlTotal = $state(0);
  let currentURL = $state('');
  let cancelStream = null;

  function handleSubmit(e) {
    e.preventDefault();
    if (!url.trim()) return;
    state = 'discovering';
    errorMsg = '';
    result = '';
    discoveredURLs = [];
    crawlDone = 0;
    crawlTotal = 0;
    currentURL = '';

    cancelStream = generateLlmsTxtStream(url.trim(), {
      onDiscovered(urls, total) {
        discoveredURLs = urls;
        crawlTotal = total;
        state = 'crawling';
      },
      onProgress(pageURL, done, total) {
        currentURL = pageURL;
        crawlDone = done;
        crawlTotal = total;
      },
      onDone(llmsTxt) {
        result = llmsTxt;
        state = 'result';
        cancelStream = null;
      },
      onError(msg) {
        errorMsg = msg;
        state = 'error';
        cancelStream = null;
      },
    });
  }

  function handleReset() {
    if (cancelStream) cancelStream();
    state = 'idle';
    result = '';
    errorMsg = '';
    url = '';
    copied = false;
    discoveredURLs = [];
    crawlDone = 0;
    crawlTotal = 0;
    currentURL = '';
    cancelStream = null;
  }

  function copyToClipboard() {
    navigator.clipboard.writeText(result);
    copied = true;
    setTimeout(() => copied = false, 2000);
  }

  function download() {
    const blob = new Blob([result], { type: 'text/plain' });
    const href = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = href;
    a.download = 'llms.txt';
    a.click();
    URL.revokeObjectURL(href);
  }
</script>

<main>
  <div class="container">
    <h1>llms.txt Generator</h1>
    <p class="subtitle">Generate an <a href="https://llmstxt.org" target="_blank" rel="noopener">llms.txt</a> file for any website</p>

    {#if state === 'idle' || state === 'error'}
      <form onsubmit={handleSubmit}>
        <div class="input-group">
          <input
            type="url"
            bind:value={url}
            placeholder="https://example.com"
            required
          />
          <button type="submit">Generate</button>
        </div>
      </form>
      {#if state === 'error'}
        <div class="error">{errorMsg}</div>
      {/if}

    {:else if state === 'discovering'}
      <div class="loading">
        <div class="spinner"></div>
        <p>Discovering pages...</p>
      </div>

    {:else if state === 'crawling'}
      <div class="crawl-progress">
        <div class="progress-header">
          <p>Found <strong>{crawlTotal}</strong> pages to crawl</p>
          <p class="progress-count">Fetching page {crawlDone} of {crawlTotal}</p>
        </div>
        <div class="progress-bar-track">
          <div class="progress-bar-fill" style="width: {crawlTotal ? (crawlDone / crawlTotal * 100) : 0}%"></div>
        </div>
        {#if currentURL}
          <p class="current-url">{currentURL}</p>
        {/if}
        <div class="url-list">
          {#each discoveredURLs as pageURL}
            <div class="url-item">{pageURL}</div>
          {/each}
        </div>
      </div>

    {:else if state === 'result'}
      <div class="actions">
        <button onclick={copyToClipboard} class="secondary">
          {copied ? 'Copied!' : 'Copy to Clipboard'}
        </button>
        <button onclick={download} class="secondary">Download llms.txt</button>
        <button onclick={handleReset} class="secondary">Generate Another</button>
      </div>
      <pre class="output"><code>{result}</code></pre>
    {/if}
  </div>
</main>
