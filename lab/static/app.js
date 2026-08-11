"use strict";

const state = { origins: [], operations: [], schemas: {}, result: null, tab: "typed" };
const byId = (id) => document.getElementById(id);

async function api(path, options = {}) {
  const response = await fetch(path, { credentials: "omit", ...options });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(payload.error?.message || `Request failed (${response.status})`);
  return payload;
}

async function boot() {
  try {
    const [status, catalog] = await Promise.all([api("/api/v1/status"), api("/api/v1/origins")]);
    state.origins = catalog.origins;
    byId("api-state").textContent = `${status.origins} ORIGINS · ${status.operations} OPERATIONS`;
    const origin = byId("origin");
    catalog.origins.forEach((item) => origin.add(new Option(`${item.title} — ${item.source_class}`, item.id)));
    origin.addEventListener("change", loadOrigin);
    byId("operation").addEventListener("change", renderInputs);
    byId("invoke-form").addEventListener("submit", invoke);
    document.querySelectorAll("[data-tab]").forEach((button) => button.addEventListener("click", selectTab));
    await loadOrigin();
  } catch (error) { showError(error.message); byId("api-state").textContent = "UNAVAILABLE"; }
}

async function loadOrigin() {
  clearError();
  const id = byId("origin").value;
  const [operations, schemas] = await Promise.all([api(`/api/v1/origins/${encodeURIComponent(id)}/operations`), api(`/api/v1/origins/${encodeURIComponent(id)}/schema`)]);
  state.operations = operations.operations;
  state.schemas = schemas.schemas;
  const select = byId("operation"); select.replaceChildren();
  state.operations.forEach((item) => select.add(new Option(item.title, item.id)));
  renderInputs();
}

function renderInputs() {
  const operationId = byId("operation").value;
  const operation = state.operations.find((item) => item.id === operationId);
  const schema = state.schemas[operationId] || { properties: {}, required: [] };
  const container = byId("typed-inputs"); container.replaceChildren();
  Object.entries(schema.properties || {}).forEach(([name, property]) => {
    const label = document.createElement("label"); label.htmlFor = `input-${name}`; label.textContent = name.replaceAll("_", " ");
    let control;
    if (property.enum) { control = document.createElement("select"); property.enum.forEach((value) => control.add(new Option(String(value), String(value)))); }
    else { control = document.createElement("input"); control.type = property.type === "integer" ? "number" : "text"; }
    control.id = `input-${name}`; control.name = name; control.dataset.type = property.type; control.required = (schema.required || []).includes(name); control.setAttribute("aria-describedby", `help-${name}`);
    const help = document.createElement("small"); help.id = `help-${name}`; help.textContent = property.description || "Bounded typed input";
    container.append(label, control, help);
  });
  const selectedOrigin = state.origins.find((item) => item.id === byId("origin").value);
  byId("schema").href = `/api/v1/origins/${encodeURIComponent(selectedOrigin?.id || "")}/schema`;
  const fresh = selectedOrigin?.fresh_enabled ? "enabled" : "disabled";
  byId("operation-notes").textContent = `${operation?.description || ""} Effect: read. Admission: ${selectedOrigin?.admission_status || "unknown"}. Fresh mode: ${fresh}. Health: ${selectedOrigin?.health_state || "not_probed"}. ${selectedOrigin?.attribution || ""}`;
}

async function invoke(event) {
  event.preventDefault(); clearError();
  const button = byId("invoke"); button.disabled = true; button.firstChild.textContent = "Invoking… "; byId("result-state").textContent = "VERIFYING";
  try {
    const input = {};
    byId("typed-inputs").querySelectorAll("input, select").forEach((control) => { input[control.name] = control.dataset.type === "integer" ? Number(control.value) : control.value; });
    state.result = await api("/api/v1/invoke", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ origin_id: byId("origin").value, operation_id: byId("operation").value, mode: byId("mode").value, input }) });
    byId("empty-result").hidden = true; byId("result").hidden = false; byId("result-state").textContent = state.result.status.toUpperCase(); byId("result-id").textContent = state.result.result_id;
    byId("bundle").href = `/api/v1/results/${encodeURIComponent(state.result.result_id)}/bundle`; state.tab = "typed"; updateTabs(); renderResult();
  } catch (error) { showError(error.message); byId("result-state").textContent = "REJECTED"; }
  finally { button.disabled = false; button.firstChild.textContent = "Invoke typed operation "; }
}

function selectTab(event) { state.tab = event.currentTarget.dataset.tab; updateTabs(); renderResult(); }
function updateTabs() { document.querySelectorAll("[data-tab]").forEach((button) => button.setAttribute("aria-selected", String(button.dataset.tab === state.tab))); }
function renderResult() {
  const result = state.result; if (!result) return;
  const views = {
    typed: { operation: result.operation_id, status: result.status, result: Object.fromEntries(result.fields.map((field) => [field.id, field.semantic.lexical ?? null])) },
    native: Object.fromEntries(result.fields.map((field) => [field.id, { term: field.native.term, locator: field.native.locator, lexical: field.native.lexical ?? null }])),
    transformation: Object.fromEntries(result.fields.map((field) => [field.id, { native: field.native.lexical ?? null, transformations: field.derivation.transforms, mapping: field.derivation.mapping, semantic: field.semantic.lexical ?? null }])),
    evidence: { observed_at: result.observed_at, result_digest: result.result_digest, bundle_id: result.bundle_id, bindings: result.bindings },
    verify: { go: `bin/twirx-lab verify --root . --bundle <downloaded-bundle-directory>`, c: `bin/tw-verify-result-c <downloaded-bundle-directory>`, result_id: result.result_id, note: "Verification is offline after the proof bundle is downloaded." }
  };
  byId("result-content").textContent = JSON.stringify(views[state.tab], null, 2);
}
function showError(message) { const element = byId("error"); element.textContent = message; element.hidden = false; }
function clearError() { byId("error").hidden = true; byId("error").textContent = ""; }
document.addEventListener("DOMContentLoaded", boot);
