<script>
  import { generateLlmsTxt } from './lib/api.js';

  let state = $state('idle');
  let result = $state('');
  let errorMsg = $state('');
  let url = $state('');
  let copied = $state(false);

  async function handleSubmit(e) {
    e.preventDefault();
    if (!url.trim()) return;
    state = 'loading';
    errorMsg = '';
    try {
      result = await generateLlmsTxt(url.trim());
      state = 'result';
    } catch (err) {
      errorMsg = err.message;
      state = 'error';
    }
  }

  function handleReset() {
    state = 'idle';
    result = '';
    errorMsg = '';
    url = '';
    copied = false;
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
            autofocus
          />
          <button type="submit">Generate</button>
        </div>
      </form>
      {#if state === 'error'}
        <div class="error">{errorMsg}</div>
      {/if}

    {:else if state === 'loading'}
      <div class="loading">
        <div class="spinner"></div>
        <p>Crawling site and generating llms.txt...</p>
        <p class="loading-detail">This may take a moment depending on the site size.</p>
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
