(function(){
  var form=$('#form'),save=$('#save'),state=$('#save-state'),publicLink=$('#public_sharing_enabled'),embedding=$('#embedding_enabled'),embeddingRow=$('#embedding-row'),impact=$('#public-impact');
  var original={},publicBoards=0,saving=false;
  function values(){return {site_name:$('#site_name').value.trim(),timezone:$('#timezone').value,public_sharing_enabled:publicLink.checked,embedding_enabled:embedding.checked};}
  function changed(){var current=values();return Object.keys(current).some(function(key){return current[key]!==original[key];});}
  function syncAccess(){var enabled=publicLink.checked;embedding.disabled=!enabled;embeddingRow.classList.toggle('is-disabled',!enabled);if(!enabled)embedding.checked=false;impact.hidden=enabled||!publicBoards;impact.textContent=publicBoards?'关闭后，'+publicBoards+' 个已发布仪表盘的公开链接将立即不可访问。':'';}
  function syncState(){syncAccess();if(saving)return;state.textContent=changed()?'有未保存的更改':'已同步';state.className='save-state'+(changed()?' dirty':'');}
  function fill(settings){original={site_name:settings.site_name||'Topbase',timezone:settings.timezone||'Asia/Shanghai',public_sharing_enabled:!!settings.public_sharing_enabled,embedding_enabled:!!settings.embedding_enabled};$('#site_name').value=original.site_name;$('#timezone').value=original.timezone;publicLink.checked=original.public_sharing_enabled;embedding.checked=original.embedding_enabled;syncState();}
  function load(){return Promise.all([api('/api/admin/settings'),api('/api/dashboards').catch(function(){return []})]).then(function(result){publicBoards=(result[1]||[]).filter(function(board){return board.public_uuid;}).length;fill(result[0]);}).catch(function(error){toast(error.message);});}
  form.addEventListener('input',syncState);form.addEventListener('change',syncState);
  form.addEventListener('submit',function(event){event.preventDefault();if(saving||!form.reportValidity())return;saving=true;save.disabled=true;state.textContent='正在保存…';state.className='save-state saving';api('/api/admin/settings','PUT',values()).then(function(settings){fill(settings);toast('工作区设置已保存');}).catch(function(error){toast(error.message);}).finally(function(){saving=false;save.disabled=false;syncState();});});
  load();
}());
