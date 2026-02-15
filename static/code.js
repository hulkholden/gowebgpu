function showError(msg) {
  const el = document.getElementById("error");
  if (el) {
    el.textContent = msg;
    el.style.display = "block";
  }
}

async function init() {
  if (!navigator.gpu) {
    showError("WebGPU not supported in this browser.");
    return;
  }

  const adapter = await navigator.gpu.requestAdapter();
  if (!adapter) {
    showError("Couldn't request WebGPU adapter.");
    return;
  }
  const device = await adapter.requestDevice();

  const canvas = document.querySelector("#display");
  const context = canvas.getContext("webgpu");

  context.configure({
    device: device,
    format: navigator.gpu.getPreferredCanvasFormat(),
	 	alphaMode: "premultiplied",
  });

  window.getDevice = () => {
    return device;
  };
  window.getContext = () => {
    return context;
  };
  window.getExample = () => {
    const params = new URLSearchParams(window.location.search);
    return params.get("example") || "battle";
  };
}

// Expose showError to Go/WASM so it can display errors in the UI.
window.showError = showError;

// Capture unhandled errors (e.g. WASM panics).
window.addEventListener("error", (e) => {
  showError("Error: " + e.message);
});
window.addEventListener("unhandledrejection", (e) => {
  showError("Error: " + (e.reason?.message || e.reason));
});

init().catch((e) => showError("Init error: " + e.message));