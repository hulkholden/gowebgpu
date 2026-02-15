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
}

init().catch((e) => showError("Init error: " + e.message));