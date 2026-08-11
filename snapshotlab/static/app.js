"use strict";

const snapshotID = "sha256:54739822257ef617b136454285a8fd47802f0960c7cf53a49abd2d5d1f1389c5";

const presets = {
  "cross-origin": {
    label: "TWIRX and World Bank current semantic view",
    path: "/queries/cross-origin.json"
  },
  "archive-history": {
    label: "RFC Editor observed-native title history",
    path: "/queries/archive-history.json"
  }
};

let selectedPreset = "cross-origin";
let atlasOrigins = [];
const byID = (id) => document.getElementById(id);

async function requestJSON(path, options = {}) {
  const response = await fetch(path, { credentials: "omit", ...options });
  const value = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(value.error || `Request failed (${response.status})`);
  return value;
}

function shortDigest(value) {
  if (!value || value.length < 28) return value || "—";
  return `${value.slice(0, 18)}…${value.slice(-10)}`;
}

async function boot() {
  byID("snapshot-id").textContent = `SNAPSHOT ${shortDigest(snapshotID)}`;
  document.querySelectorAll("[data-preset]").forEach((button) => button.addEventListener("click", () => selectPreset(button.dataset.preset)));
  byID("run-query").addEventListener("click", runQuery);
  byID("origin-search").addEventListener("input", renderOrigins);
  byID("family-filter").addEventListener("change", renderOrigins);
  try {
    const [status, deltas, origins] = await Promise.all([requestJSON("/api/v1/status"), requestJSON("/api/v1/deltas?limit=10"), requestJSON("/api/v1/origins?limit=500")]);
    if (status.snapshot_id !== snapshotID || status.execution !== "immutable_materialized_snapshot_only") throw new Error("Unexpected snapshot identity or execution mode");
    if (origins.total !== status.actual.atlas_identities || origins.items.length !== origins.total) throw new Error("Atlas catalog does not reconcile with the snapshot");
    byID("service-state").textContent = "VERIFIED SNAPSHOT ONLINE";
    byID("atlas-count").textContent = status.actual.atlas_identities;
    byID("packet-count").textContent = status.actual.public_packets;
    byID("origin-count").textContent = `across ${status.actual.public_origins_with_packets} origins`;
    byID("view-count").textContent = status.actual.materialized_views;
    byID("delta-count").textContent = status.actual.deltas;
    atlasOrigins = origins.items;
    populateFamilies();
    renderOrigins();
    renderDeltas(deltas.items || []);
  } catch (error) {
    byID("service-state").textContent = "UNAVAILABLE";
    showError(error.message);
  }
}

function populateFamilies() {
  const select = byID("family-filter");
  [...new Set(atlasOrigins.map((origin) => origin.domain_family))].sort().forEach((family) => {
    const option = document.createElement("option");
    option.value = family;
    option.textContent = family.replaceAll("_", " ");
    select.append(option);
  });
}

function renderOrigins() {
  const needle = byID("origin-search").value.trim().toLowerCase();
  const family = byID("family-filter").value;
  const visible = atlasOrigins.filter((origin) => {
    const searchable = `${origin.origin_id} ${origin.canonical_host} ${origin.domain_family}`.toLowerCase();
    return (!needle || searchable.includes(needle)) && (!family || origin.domain_family === family);
  });
  const packetBearing = visible.filter((origin) => origin.public_packet_count > 0).length;
  byID("atlas-summary").textContent = `${visible.length} of ${atlasOrigins.length} selected identities shown · ${packetBearing} shown identities have public packets in this snapshot`;
  const body = byID("origins");
  body.replaceChildren();
  visible.forEach((origin) => {
    const row = document.createElement("tr");
    const id = document.createElement("th");
    id.scope = "row";
    id.textContent = origin.origin_id;
    const host = document.createElement("td");
    host.textContent = origin.canonical_host;
    const familyCell = document.createElement("td");
    familyCell.textContent = origin.domain_family.replaceAll("_", " ");
    const state = document.createElement("td");
    const badge = document.createElement("span");
    badge.className = origin.public_packet_count > 0 ? "packet-state available" : "packet-state";
    badge.textContent = origin.public_packet_count > 0 ? `${origin.public_packet_count} public packets` : "selected · no public packets";
    state.append(badge);
    row.append(id, host, familyCell, state);
    body.append(row);
  });
}

function selectPreset(name) {
  if (!presets[name]) return;
  selectedPreset = name;
  document.querySelectorAll("[data-preset]").forEach((button) => button.classList.toggle("active", button.dataset.preset === name));
}

async function runQuery() {
  const button = byID("run-query");
  button.disabled = true;
  byID("result-state").textContent = "QUERYING IMMUTABLE STATE";
  clearError();
  try {
    const query = await requestJSON(presets[selectedPreset].path);
    const result = await requestJSON("/api/v1/query", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(query)
    });
    if (result.snapshot_id !== snapshotID || result.plan.network_requests !== 0) throw new Error("Runtime returned an unexpected identity or network plan");
    renderResult(result);
  } catch (error) {
    showError(error.message);
    byID("result-state").textContent = "REJECTED";
  } finally {
    button.disabled = false;
  }
}

function renderResult(result) {
  byID("empty-result").hidden = true;
  byID("result").hidden = false;
  byID("result-state").textContent = result.status.toUpperCase();
  byID("query-digest").textContent = shortDigest(result.query_digest);
  byID("result-digest").textContent = shortDigest(result.result_digest);
  byID("network-requests").textContent = result.plan.network_requests;
  byID("fixtures-excluded").textContent = result.plan.excluded_fixtures;
  byID("result-caption").textContent = `${presets[selectedPreset].label} — ${result.rows.length} rows`;
  const body = byID("rows");
  body.replaceChildren();
  result.rows.forEach((row) => body.append(rowElement(row)));
  byID("raw-result").textContent = JSON.stringify({ snapshot_id: result.snapshot_id, status: result.status, query_digest: result.query_digest, plan_digest: result.plan_digest, result_digest: result.result_digest, plan: result.plan, economic_event: result.economic_event }, null, 2);
}

function rowElement(row) {
  const tr = document.createElement("tr");
  const origin = document.createElement("th");
  origin.scope = "row";
  origin.textContent = row.origin_id;
  const native = document.createElement("td");
  const nativeValue = document.createElement("b");
  nativeValue.textContent = row.native_lexical;
  const nativeMeta = document.createElement("small");
  nativeMeta.textContent = `${row.native_term} · ${row.native_locator}`;
  native.append(nativeValue, nativeMeta);
  const semantic = document.createElement("td");
  semantic.textContent = row.semantic_term || "observed-native only";
  const proof = document.createElement("td");
  const trace = document.createElement("button");
  trace.type = "button";
  trace.className = "tracebutton";
  trace.textContent = "Trace";
  trace.addEventListener("click", () => showTrace(row.packet_digest));
  const packet = document.createElement("a");
  packet.href = `/api/v1/packets/${row.packet_digest.replace("sha256:", "")}`;
  packet.textContent = "CBOR";
  packet.setAttribute("download", "semantic-packet.cbor");
  proof.append(trace, packet);
  tr.append(origin, native, semantic, proof);
  return tr;
}

async function showTrace(digest) {
  const dialog = byID("trace-dialog");
  byID("trace-content").textContent = "Loading trace…";
  dialog.showModal();
  try {
    const trace = await requestJSON(`/api/v1/trace/${digest.replace("sha256:", "")}`);
    byID("trace-content").textContent = JSON.stringify(trace, null, 2);
  } catch (error) {
    byID("trace-content").textContent = error.message;
  }
}

function renderDeltas(items) {
  const root = byID("deltas");
  root.replaceChildren();
  if (!items.length) {
    const empty = document.createElement("p");
    empty.textContent = "No admitted deltas in this snapshot.";
    root.append(empty);
    return;
  }
  items.forEach((delta) => {
    const article = document.createElement("article");
    const heading = document.createElement("h3");
    heading.textContent = `${delta.class.toUpperCase()} DELTA · ${delta.kind}`;
    const summary = document.createElement("p");
    summary.textContent = `${delta.origin_id} · ${delta.reason_code} · ${delta.occurred_at}`;
    const chain = document.createElement("p");
    chain.className = "digestline";
    chain.textContent = `${shortDigest(delta.before_packet_digest)} → ${shortDigest(delta.after_packet_digest)}`;
    const download = document.createElement("a");
    download.href = `/api/v1/deltas/${delta.digest.replace("sha256:", "")}`;
    download.textContent = "Download canonical delta (CBOR)";
    download.setAttribute("download", "semantic-delta.cbor");
    article.append(heading, summary, chain, download);
    root.append(article);
  });
}

function showError(message) {
  const error = byID("error");
  error.textContent = message;
  error.hidden = false;
}

function clearError() {
  byID("error").textContent = "";
  byID("error").hidden = true;
}

document.addEventListener("DOMContentLoaded", boot);
