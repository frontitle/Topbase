(function () {
  function init() {
    var sections = [].slice.call(document.querySelectorAll('.notify-page .section-block'));
    if (sections.length < 3 || !window.TopbaseTabs) return setTimeout(init, 120);
    var labels = ['Webhook 通道', '订阅管理', '发送记录'];
    var tabs = document.createElement('nav');
    tabs.className = 'tb-tabs notify-tabs';
    labels.forEach(function (label, index) {
      var button = document.createElement('button');
      button.type = 'button';
      button.dataset.tab = String(index);
      button.textContent = label;
      tabs.appendChild(button);
      sections[index].dataset.panel = String(index);
    });
    document.querySelector('.notify-page .page-head').after(tabs);
    var controller = TopbaseTabs.mount(tabs, {
      initial: '0',
      onChange: function (index) {
        var hasHooks = !!document.querySelector('#hooks .hook-card');
        if (Number(index) > 0 && !hasHooks) {
          toast('请先创建并启用至少一个 Webhook 通道');
          controller.activate('0', false);
        }
      }
    });
    new MutationObserver(function () {
      if (!document.querySelector('#hooks .hook-card') && controller.value() !== '0') controller.activate('0', false);
    }).observe(document.querySelector('#hooks'), { childList: true, subtree: true });
  }
  setTimeout(init, 160);
}());
