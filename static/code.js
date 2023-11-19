async function init() {
  if (!navigator.gpu) {
    throw Error("WebGPU not supported.");
  }

  const adapter = await navigator.gpu.requestAdapter();
  if (!adapter) {
    throw Error("Couldn't request WebGPU adapter.");
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

init();