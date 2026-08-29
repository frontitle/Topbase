(function(){
  api('/api/version?check=1').then(function(info){
    var current=info.version||'—',latest=info.latest_version||current;
    $('#current-version').textContent=current;
    if(latest===current)return;
    var card=$('#update-available');
    $('#latest-version').textContent=latest;
    $('#latest-state').textContent=info.update_notes||'服务器检测到新版本，可查看升级说明了解功能变更。';
    if(info.upgrade_url){var link=$('#upgrade-link');link.href=info.upgrade_url;link.hidden=false}
    card.hidden=false;
  }).catch(function(e){
    $('#current-version').textContent='暂时无法检查';
    toast(e.message);
  });
}());
