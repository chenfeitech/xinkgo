(function () {
  "use strict";
  var shim = window.__XINKGO_SHIM__;
  if (!shim || !shim.boot) return;
  shim.boot().catch(function (err) {
    shim.fail(String((err && err.stack) || err));
  });
})();
