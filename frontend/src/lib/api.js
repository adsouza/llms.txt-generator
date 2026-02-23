export async function generateLlmsTxt(url) {
  const resp = await fetch('/api/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url }),
  });

  const data = await resp.json();

  if (!resp.ok) {
    throw new Error(data.detail || data.title || 'Generation failed');
  }

  return data.llms_txt;
}
