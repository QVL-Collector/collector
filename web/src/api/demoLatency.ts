const demoLatencyMs = 450;

function abortError(): DOMException {
  return new DOMException('Aborted', 'AbortError');
}

export function waitForDemoLatency(signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted === true) {
      reject(abortError());
      return;
    }
    const timer = window.setTimeout(resolve, demoLatencyMs);
    signal?.addEventListener('abort', () => {
      window.clearTimeout(timer);
      reject(abortError());
    });
  });
}
