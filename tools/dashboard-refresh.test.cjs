const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "../backend/cmd/map/templates/static/js/dashboard.js"), "utf8");

test("dashboard refresh replaces the snapshot, reports failures, and recovers", async () => {
  let snapshot = "initial";
  let pending;
  let response = { ok: true, text: async () => "live competition" };
  let requests = 0;
  const notice = { hidden: true };
  const document = {
    hidden: false,
    getElementById: (id) => id === "dashboard-refresh-error" ? notice : {
      replaceWith: (next) => { snapshot = next; },
    },
  };
  vm.runInNewContext(source, {
    document,
    AbortSignal,
    window: {
      location: { href: "http://localhost/test/admin" },
      setTimeout: (callback, delay) => {
        assert.equal(delay, 15000);
        pending = callback;
      },
    },
    fetch: async (url, options) => {
      requests++;
      assert.equal(url, "http://localhost/test/admin");
      assert.equal(options.cache, "no-store");
      return response;
    },
    DOMParser: class {
      parseFromString(html) {
        return { getElementById: () => html === "login page" ? null : html };
      }
    },
  });

  await pending();
  assert.equal(snapshot, "live competition");
  assert.equal(notice.hidden, true);

  for (const failure of [
    { ok: false },
    { ok: true, redirected: true },
    { ok: true, text: async () => "login page" },
    { ok: true, text: async () => { throw new Error("offline"); } },
  ]) {
    response = failure;
    await pending();
    assert.equal(snapshot, "live competition");
    assert.equal(notice.hidden, false);
  }

  response = { ok: true, text: async () => "no ongoing competition" };
  await pending();
  assert.equal(snapshot, "no ongoing competition");
  assert.equal(notice.hidden, true);

  document.hidden = true;
  const before = requests;
  await pending();
  assert.equal(requests, before);
});
