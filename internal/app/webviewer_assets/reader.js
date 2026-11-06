// A small readability-style extractor the Web viewer runs with Page.Eval.
// It is deliberately heuristic: no vendored Readability, no new runtime
// dependency. It returns the fields Go needs for reader view and saving.
(function () {
  window.__xinkgoWebExtract = function () {
    function esc(text) {
      return String(text || '').replace(/[&<>"']/g, function (ch) {
        return { '&': '&amp;', '<': '&lt;', '>': '&gt;',
                 '"': '&quot;', "'": '&#39;' }[ch];
      });
    }
    var title = document.title || location.hostname || 'page';
    var candidates = [
      document.querySelector('article'),
      document.querySelector('[role="main"]'),
      document.querySelector('main'),
      document.querySelector('.post-content, .entry-content, .article-content')
    ];
    var root = null;
    for (var i = 0; i < candidates.length; i++) {
      if (candidates[i] && (candidates[i].innerText || '').length > 200) {
        root = candidates[i];
        break;
      }
    }
    if (!root) {
      var best = null, score = 0;
      var nodes = document.querySelectorAll('div, section');
      for (var j = 0; j < nodes.length; j++) {
        var size = (nodes[j].innerText || '').length;
        if (size > score && size < 200000) {
          best = nodes[j];
          score = size;
        }
      }
      root = best || document.body;
    }
    var clone = root.cloneNode(true);
    var drop = clone.querySelectorAll(
      'script, style, nav, aside, header, footer, form, iframe, ' +
      'noscript, svg, .ad, .ads, .advertisement, [aria-hidden="true"]');
    for (var k = 0; k < drop.length; k++) {
      if (drop[k].parentNode) drop[k].parentNode.removeChild(drop[k]);
    }
    var text = (clone.innerText || clone.textContent || '').trim();
    var html = '<!doctype html><html><head><meta charset="utf-8">' +
      '<base href="' + esc(location.href) + '">' +
      '<style>body{font-family:system-ui,sans-serif;line-height:1.6;' +
      'max-width:44em;margin:2em auto;padding:0 1em;color:inherit}' +
      'img{max-width:100%}pre{white-space:pre-wrap}' +
      '</style></head><body><h1>' + esc(title) + '</h1>' +
      clone.innerHTML + '</body></html>';
    return { title: title, text: text, html: html, url: location.href };
  };
})();
