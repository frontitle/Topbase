let testPassed=false, databases=[], current=null, tables=[], activeTable=null, editingId=null;
const engineProfiles={
  postgres:{label:'PostgreSQL',port:5432,network:true,username:true,ssh:true,dsn:'postgres://user:password@host:5432/database?sslmode=require',hint:'标准 PostgreSQL 协议。'},
  mysql:{label:'MySQL / MariaDB',port:3306,network:true,username:true,ssh:true,dsn:'user:password@tcp(host:3306)/database?parseTime=true',hint:'同时兼容 MariaDB、TiDB、OceanBase MySQL 模式、Doris 和 StarRocks。'},
  clickhouse:{label:'ClickHouse',port:9000,network:true,username:false,ssh:true,defaultUser:'default',dsn:'clickhouse://default:password@host:9000/database?secure=true',hint:'使用 ClickHouse 原生 TCP 协议。用户名留空时使用 default。'},
  sqlserver:{label:'SQL Server',port:1433,network:true,username:true,ssh:true,dsn:'sqlserver://user:password@host:1433?database=database&encrypt=true',hint:'兼容 SQL Server、Azure SQL Database 和 Azure Synapse SQL。'},
  oracle:{label:'Oracle Database',port:1521,network:true,username:true,ssh:true,dsn:'oracle://user:password@host:1521/service_name',hint:'使用纯 Go 驱动连接 Oracle Service Name，无需安装 Instant Client。'},
  sqlite:{label:'SQLite',port:0,network:false,username:false,ssh:false,dsn:'/data/analytics.db',hint:'填写 Topbase 服务器能够访问的 SQLite 文件绝对路径。'}
};
function profile(){return engineProfiles[$('#engine').value]||engineProfiles.postgres}
function engineLabel(id){return (engineProfiles[id]||{}).label||id||'数据库'}
function normalizeLabels(){$$('.form-grid label').forEach(label=>{const control=label.querySelector('input,select,textarea');if(!control||label.querySelector('.field-label'))return;const title=document.createElement('span');title.className='field-label';while(label.firstChild!==control)title.append(label.firstChild);control.classList.add('field-control');label.prepend(title)});['#port','#password','#ssl','#dsn','#ssh-port','#ssh-key-password','#ssh-fingerprint'].forEach(id=>$(id)?.closest('label')?.querySelector('em')?.remove());const sshHelp=$$('[data-panel="ssh"] .panel-help')[0];if(sshHelp)sshHelp.textContent='建议填写 SHA256 主机密钥指纹以验证跳板机身份；留空时仍可连接，但不会进行该身份校验。'}
normalizeLabels();
function labelText(control,text){const title=$(control)?.closest('label')?.querySelector('.field-label');if(title&&title.firstChild)title.firstChild.textContent=text+' '}
function applyEngineUI(resetPort){
  const p=profile();
  $$('[data-network]').forEach(el=>el.hidden=!p.network);
  $$('[data-username],[data-password]').forEach(el=>el.hidden=!p.network);
  const sshTab=$('[data-tab="ssh"]');if(sshTab)sshTab.hidden=!p.ssh;
  if(!p.ssh&&$('[data-panel="ssh"]')?.classList.contains('active')){$('[data-tab="connection"]').click()}
  labelText('#database',p.network?'数据库名称':'数据库文件');
  $('#database').placeholder=p.network?'例如：analytics':'例如：/data/analytics.db';
  if(resetPort||!$('#port').value)$('#port').value=p.port||'';
  if(p.defaultUser&&!$('#username').value)$('#username').value=p.defaultUser;
  $('#dsn').placeholder=editingId?'留空则保持原连接字符串':p.dsn;
  $('#wizard-title').textContent=(editingId?'修改 ':'连接 ')+p.label;
  $('#wizard-help').textContent=(editingId?'修改完成后重新测试并保存。':'按需填写连接与网络设置，然后测试并保存。')+' '+p.hint;
  const connectionHelp=$$('[data-panel="connection"] .panel-help')[0];if(connectionHelp)connectionHelp.textContent=p.network?'建议使用只读账号，只授予需要分析的数据表权限。':p.hint;
  const securityHelp=$$('[data-panel="security"] .panel-help')[0];if(securityHelp)securityHelp.textContent=p.network?'连接字符串会覆盖主机、端口、数据库及账号字段。':'可在连接信息中填写文件路径，也可在这里填写 SQLite DSN。';
}
function setStatus(kind,message){const e=$('#connection-status');e.className='connection-status '+kind;e.textContent=message}
function resetTest(){testPassed=false;$('#save').disabled=true;setStatus('', editingId?'修改连接后请先测试，再保存。':'尚未测试连接')}
function setWizardMode(id){
  editingId=id||null;
  $('#wizard-kicker').textContent=editingId?'编辑连接':'添加数据库';
  $('#save').textContent=editingId?'保存并重新扫描':'保存数据库';
  $('#password').placeholder=editingId?'留空则保持原密码':'输入数据库密码';
  applyEngineUI(false);
}
function clearWizard(){
  ['#name','#host','#database','#username','#password','#dsn','#ssh-host','#ssh-username','#ssh-password','#ssh-private-key','#ssh-key-password','#ssh-fingerprint'].forEach(id=>$(id).value='');
  $('#engine').value='postgres';$('#port').value='5432';$('#ssh-port').value='22';$('#ssl').value='prefer';$('#ssh-auth').value='key';
  $$('.tab').forEach(e=>e.classList.toggle('active',e.dataset.tab==='connection'));
  $$('.panel').forEach(e=>e.classList.toggle('active',e.dataset.panel==='connection'));
  syncSSHAuth();applyEngineUI(true);
}
function catalogHint(db){
  const hint={name:db.name||'',engine:db.engine||'postgres'};
  const host=db.host||'';
  const cut=host.lastIndexOf(':');
  if(cut>0 && host.indexOf(']')===-1){hint.host=host.slice(0,cut);hint.port=Number(host.slice(cut+1))||(engineProfiles[hint.engine]||engineProfiles.postgres).port}else{hint.host=host}
  return hint;
}
function fillWizard(conn){
  clearWizard();
  $('#engine').value=conn.engine||'postgres';
  $('#name').value=conn.name||'';
  $('#host').value=conn.host||'';
  $('#port').value=conn.port||(engineProfiles[conn.engine]||engineProfiles.postgres).port;
  $('#database').value=conn.database||'';
  $('#username').value=conn.username||'';
  $('#password').value=conn.password||'';
  $('#ssl').value=conn.ssl_mode||'prefer';
  $('#dsn').value=(!conn.host && conn.dsn)?conn.dsn:'';
  if(conn.ssh&&conn.ssh.host){
    $('#ssh-host').value=conn.ssh.host||'';
    $('#ssh-port').value=conn.ssh.port||22;
    $('#ssh-username').value=conn.ssh.username||'';
    $('#ssh-auth').value=conn.ssh.authentication_type||(conn.ssh.private_key?'key':'password');
    $('#ssh-password').value=conn.ssh.password||'';
    $('#ssh-private-key').value=conn.ssh.private_key||'';
    $('#ssh-key-password').value=conn.ssh.private_key_password||'';
    $('#ssh-fingerprint').value=conn.ssh.host_key_fingerprint||'';
  }
  syncSSHAuth();applyEngineUI(false);
}
function connectionData(){
  const data={name:$('#name').value,engine:$('#engine').value,host:$('#host').value,port:Number($('#port').value),database:$('#database').value,username:$('#username').value,password:$('#password').value,ssl_mode:$('#ssl').value,dsn:$('#dsn').value};
  const sshHost=$('#ssh-host').value.trim();
  if(sshHost){
    data.ssh={host:$('#ssh-host').value,port:Number($('#ssh-port').value),username:$('#ssh-username').value,authentication_type:$('#ssh-auth').value,password:$('#ssh-password').value,private_key:$('#ssh-private-key').value,private_key_password:$('#ssh-key-password').value,host_key_fingerprint:$('#ssh-fingerprint').value};
  }else if(editingId){
    data.clear_ssh=true;
  }
  return data;
}
function validate(data){
  if(!data.name.trim())return '请填写数据库显示名称。';
  const p=engineProfiles[data.engine]||engineProfiles.postgres;
  if(data.engine==='sqlite'&&!data.database.trim()&&!data.dsn.trim())return '请填写 SQLite 数据库文件路径。';
  const complete=p.network&&!data.dsn.trim()&&(!data.host.trim()||!data.database.trim()||(p.username&&!data.username.trim()));
  if(!editingId&&complete)return '请完整填写服务器、数据库和账号，或提供连接字符串。';
  if(editingId&&p.network&&data.host.trim()&&(!data.database.trim()||(p.username&&!data.username.trim()))&&!data.dsn.trim())return '请完整填写数据库和账号，或提供连接字符串。';
  if(data.ssh){
    if(!data.ssh.host.trim()||!data.ssh.username.trim())return '启用 SSH 后，请填写跳板机主机和用户名。';
    if(!editingId){
      if(data.ssh.authentication_type==='password'&&!data.ssh.password.trim())return '请选择密码认证并填写 SSH 密码。';
      if(data.ssh.authentication_type==='key'&&!data.ssh.private_key.trim())return '请选择私钥认证并填写 SSH 私钥。';
    }
  }
  return '';
}
function syncSSHAuth(){const password=$('#ssh-auth').value==='password';document.querySelector('.ssh-password').style.display=password?'grid':'none';document.querySelector('.ssh-key').style.display=password?'none':'grid';const help=$$('[data-panel="ssh"] .panel-help')[0];if(help)help.textContent=password?'密码认证仅适用于跳板机已开启 PasswordAuthentication 的情况；部分云主机仅支持私钥认证。':'请粘贴已授权给跳板机用户的私钥。服务器主机密钥指纹为选填。'}
function showWizard(){clearWizard();setWizardMode(null);resetTest();$('#wizard').showModal()}
async function showEditWizard(){
  if(!current)return;
  setWizardMode(current.id);
  let conn=catalogHint(current);
  try{conn=Object.assign(conn, await api('/api/databases/'+current.id+'/connection'))}catch(e){toast('未找到已存凭据，请核对连接。'+e.message)}
  fillWizard(conn);
  resetTest();
  $('#wizard').showModal();
}
function showList(){current=null;activeTable=null;$('#list-view').hidden=false;$('#detail-view').hidden=true;history.replaceState({},'', '/admin/')}
function syncedText(db){return db.last_synced_at?('上次扫描 '+new Date(db.last_synced_at).toLocaleString('zh-CN')+' · '+(db.table_count||0)+' 张表'):'尚未扫描结构'}
function statusLabel(db){return '已连接'}
function renderDetailMeta(){
  if(!current)return;
  $('#detail-meta').textContent=statusLabel(current)+' · '+engineLabel(current.engine||'postgres')+' · '+(current.host||'')+' · '+syncedText(current);
}
function renderTables(){
  $('#table-count').textContent=tables.length;
  $('#table-list').innerHTML=tables.map(t=>`<button class="table-item ${activeTable&&activeTable.schema===t.schema&&activeTable.name===t.name?'active':''}" data-schema="${t.schema}" data-name="${t.name}"><b>${t.display_name||t.name}</b><small>${t.schema}.${t.name} · ${(t.columns||[]).length} 个字段</small></button>`).join('')||'<p>还没有表。请先确认连接可用，再点击「同步全部结构」。</p>';
  $$('#table-list .table-item').forEach(b=>b.onclick=()=>openTable(tables.find(t=>t.schema===b.dataset.schema&&t.name===b.dataset.name)));
}
function openTable(table){
  activeTable=table;
  renderTables();
  $('#table-title').textContent=table.schema+'.'+table.name;
  $('#table-sub').textContent=(table.columns||[]).length+' 个字段';
  $('#rescan-table').hidden=false;
  $('#field-table').innerHTML=`<table class="fields-table"><thead><tr><th>字段</th><th>类型</th><th>可空</th><th>说明</th></tr></thead><tbody>${(table.columns||[]).map(c=>`<tr><td>${c.name}${c.primary_key?' <small>主键</small>':''}</td><td>${c.data_type}</td><td>${c.nullable?'是':'否'}</td><td>${c.description||'—'}</td></tr>`).join('')}</tbody></table>`;
}
async function openDatabase(id){
  const payload=await api('/api/databases/'+id);
  current=payload.database;
  tables=payload.tables||[];
  $('#list-view').hidden=true;
  $('#detail-view').hidden=false;
  $('#detail-name').textContent=current.name;
  renderDetailMeta();
  $('#rescan-table').hidden=true;
  $('#table-title').textContent='选择一张表';
  $('#table-sub').textContent='查看字段类型，或重新扫描这一张表。';
  $('#field-table').innerHTML='';
  activeTable=null;
  renderTables();
  history.replaceState({},'', '/admin/?id='+encodeURIComponent(id));
}
async function load(){
  try{
    databases=await api('/api/databases');
    const q=($('#search').value||'').toLowerCase();
    const items=databases.filter(d=>(d.name+' '+(d.host||'')+' '+engineLabel(d.engine)).toLowerCase().includes(q));
    $('#empty').style.display=databases.length?'none':'block';
    $('#list').style.display=databases.length?'block':'none';
    $('#count').textContent=databases.length+' 个数据库';
    $('#cards').innerHTML=items.map(d=>`<article class="db-card" data-open="${d.id}"><div class="db-name"><span class="db-symbol">▣</span><div><b>${d.name}</b><small>${engineLabel(d.engine)} · ${d.host||''} · ${syncedText(d)}</small></div></div><div><span class="db-status ${d.connected===false?'off':''}">${d.connected===false?'连接中断':'已连接'}</span></div></article>`).join('');
    $$('[data-open]').forEach(card=>card.onclick=()=>openDatabase(card.dataset.open).catch(e=>toast(e.message)));
    const id=new URLSearchParams(location.search).get('id');
    if(id && databases.some(d=>d.id===id)) await openDatabase(id);
  }catch(e){toast(e.message)}
}
$('#show-form').onclick=showWizard;$('#empty-add').onclick=showWizard;
$('#search').oninput=load;
$('#back-list').onclick=()=>{showList();load()};
$('#edit-connection').onclick=()=>showEditWizard().catch(e=>toast(e.message));
$('#sync-schema').onclick=async()=>{if(!current)return;try{$('#sync-schema').disabled=true;const snap=await api('/api/databases/'+current.id+'/sync','POST',{});tables=snap.tables||[];current.last_synced_at=snap.synced_at;current.table_count=tables.length;current.connected=true;renderDetailMeta();activeTable=null;renderTables();toast('已扫描 '+tables.length+' 张表')}catch(e){toast(e.message+'  可点「编辑连接」修正主机、库名、账号、SSL 或 SSH 后再同步。')}finally{$('#sync-schema').disabled=false}};
$('#rescan-table').onclick=async()=>{if(!current||!activeTable)return;try{const snap=await api(`/api/databases/${current.id}/tables/${encodeURIComponent(activeTable.schema)}/${encodeURIComponent(activeTable.name)}/rescan`,'POST',{});tables=snap.tables||[];const next=tables.find(t=>t.schema===activeTable.schema&&t.name===activeTable.name);if(next)openTable(next);toast('已重新扫描 '+activeTable.schema+'.'+activeTable.name)}catch(e){toast(e.message)}};
$('#remove-db').onclick=async()=>{if(!current||!await confirmDialog({kicker:'数据源管理',title:'移除“'+current.name+'”？',description:'只会删除 Topbase 中保存的连接和元数据，不会删除源数据库中的任何数据。',confirmText:'移除数据源',tone:'danger'}))return;try{await api('/api/databases/'+current.id,'DELETE');toast('已移除数据源');showList();load()}catch(e){toast(e.message)}};
$$('.tab').forEach(tab=>tab.onclick=()=>{$$('.tab').forEach(e=>e.classList.toggle('active',e===tab));$$('.panel').forEach(e=>e.classList.toggle('active',e.dataset.panel===tab.dataset.tab))});
$('#ssh-auth').onchange=()=>{syncSSHAuth();resetTest()};syncSSHAuth();
$$('#wizard input,#wizard select,#wizard textarea').forEach(e=>{e.addEventListener('input',resetTest);e.addEventListener('change',resetTest)});
$('#engine').addEventListener('change',()=>applyEngineUI(true));
$('#wizard').addEventListener('close',()=>{editingId=null;$('#save').disabled=true;$('#save').textContent='保存数据库'});
$('#test').onclick=async()=>{const b=$('#test'),data=connectionData(),message=validate(data);if(message){setStatus('error',message);return}try{b.disabled=true;b.textContent='测试中…';setStatus('testing','正在验证网络、SSH 隧道和数据库权限…');const path=editingId?'/api/databases/'+editingId+'/test':'/api/databases/test';await api(path,'POST',data);testPassed=true;$('#save').disabled=false;setStatus('success','连接验证成功。现在可以保存。')}catch(err){testPassed=false;$('#save').disabled=true;setStatus('error','连接失败：'+err.message)}finally{b.disabled=false;b.textContent='测试连接'}};
$('#save').onclick=async e=>{e.preventDefault();if(!testPassed){setStatus('error','请先测试连接成功，再保存。');return}const b=$('#save'),updating=!!editingId;try{b.disabled=true;b.textContent=updating?'正在保存并扫描…':'正在保存…';const saved=updating?await api('/api/databases/'+editingId,'PUT',connectionData()):await api('/api/databases','POST',connectionData());$('#wizard').close();toast(updating?('连接已更新'+(saved.table_count?`，已扫描 ${saved.table_count} 张表`:'，请再点同步结构')):('数据库已保存'+(saved.table_count?`，已扫描 ${saved.table_count} 张表`:'，正在等待同步')));await load();if(saved.id) await openDatabase(saved.id)}catch(err){setStatus('error','保存失败：'+err.message);b.disabled=false;b.textContent=updating?'保存并重新扫描':'保存数据库'}};
load();
