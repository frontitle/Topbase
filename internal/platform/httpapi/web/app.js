function currentDatabaseID(){return $('#database').value || (window.topbasePickDatabase?topbasePickDatabase(databases):'')}
let databases=[];
function draw(rows){
  const c=$('#chart'),x=c.getContext('2d'),w=c.width,h=c.height;
  x.clearRect(0,0,w,h);
  const v=rows.map(r=>Number(r[1])).filter(Number.isFinite);
  if(!v.length)return;
  const a=Math.max(...v),b=Math.min(...v),p=18;
  x.strokeStyle='#1d1d1f';x.lineWidth=2;x.beginPath();
  v.forEach((n,i)=>{const px=p+i*(w-2*p)/Math.max(v.length-1,1),py=h-p-(n-b)/Math.max(a-b,1)*(h-2*p);i?x.lineTo(px,py):x.moveTo(px,py)});
  x.stroke();
}
async function loadTables(){
  const id=currentDatabaseID();
  if(!id){$('#schema').textContent='还没有数据源。管理员可在管理后台添加数据库。';return}
  try{
    const tables=await api('/api/databases/'+id+'/tables');
    $('#schema').innerHTML=tables.slice(0,12).map(t=>'<b>'+esc(t.schema+'.'+t.name)+'</b><br><span>'+esc(t.columns.map(c=>c.name+': '+c.data_type).join(' · '))+'</span>').join('<hr>')||'这个数据源里还没有表。';
  }catch(e){$('#schema').textContent=e.message}
}
async function loadDatabases(){
  try{
    databases=await api('/api/databases');
    const s=$('#database');
    if(!databases.length){
      s.innerHTML='<option value="">暂无数据源</option>';
      s.hidden=false;
      $('#result-grid').innerHTML='<p class="page-meta">还没有数据源。管理员可在管理后台添加数据库。</p>';
      $('#schema').textContent='添加数据源后，这里会显示可查询的表。';
      return;
    }
    const id=topbasePickDatabase(databases, s.value);
    s.innerHTML=databases.map(d=>'<option value="'+d.id+'">'+esc(d.name)+(databases.length>1?' · '+(d.engine||'postgres'):'')+'</option>').join('');
    s.value=id;
    s.hidden=databases.length===1;
    topbaseRememberDatabase(id);
    s.onchange=()=>{topbaseRememberDatabase(s.value);loadTables()};
    await loadTables();
  }catch(e){toast(e.message)}
}
async function loadQuestions(){
  try{
    const items=await api('/api/questions');
    const boards=await api('/api/dashboards');
    const notes=await api('/api/notifications').catch(()=>[]);
    const parts=[];
    if(boards.length)parts.push('<b>仪表盘</b>'+boards.map(d=>'<a href="/dashboard/'+d.id+'/">'+esc(d.name)+'</a>').join(''));
    if(items.length)parts.push('<b>分析</b>'+items.map(q=>'<a href="/questions/'+q.id+'/">'+esc(q.name)+'<small>'+esc(q.query_type||'')+'</small></a>').join(''));
    if(notes.length)parts.push('<b>站内通知</b>'+notes.slice(0,5).map(n=>'<div>'+esc(n.title)+' · '+esc(n.body||'')+'</div>').join(''));
    $('#questions').innerHTML=parts.join('')||emptyHTML({icon:'◇',title:'尚未保存分析',body:'打开数据浏览查看一张表，或在左侧运行 SQL 后保存。',href:'/questions/',cta:'打开分析列表'});
  }catch(e){$('#questions').textContent=e.message}
}
async function boot(){
  try{
    const status=await api('/api/setup/status');
    if(!status.completed){location.replace('/setup/');return}
  }catch(e){toast(e.message);return}
  try{
    const user=await api('/api/user/current');
    $('#auth-link').textContent=user.name||user.email;
    $('#auth-link').href='#';
    $('#auth-link').onclick=async ev=>{ev.preventDefault();await fetch('/api/session',{method:'DELETE'});location.reload()};
  }catch(_){
    $('#auth-link').textContent='登录';
    $('#auth-link').href='/auth/login/';
  }
  loadDatabases();
  loadQuestions();
}
$('#ask').onclick=async()=>{
  const id=currentDatabaseID();
  if(!id)return toast('还没有数据源，请先在管理后台添加');
  try{
    const d=await api('/api/ai/chat','POST',{message:$('#question').value,database_id:id});
    $('#sql').value=d.sql;
    $('#result-grid').innerHTML='<p class="page-meta">'+esc(d.answer)+'</p>';
    toast('已生成可审查的只读查询');
  }catch(e){toast(e.message)}
};
$('#run').onclick=async()=>{
  const id=currentDatabaseID();
  if(!id)return toast('还没有数据源，请先在管理后台添加');
  try{
    const d=await api('/api/queries/run','POST',{database_id:id,sql:$('#sql').value});
    TopbaseGrid('#result-grid',{columns:d.columns||[], rows:d.rows||[]});
    draw(d.rows);
    toast('查询已完成');
  }catch(e){toast(e.message)}
};
$('#save').onclick=async()=>{
  const id=currentDatabaseID();
  if(!id)return toast('还没有数据源，请先在管理后台添加');
  const sql=$('#sql').value.trim();
  if(!sql)return toast('先写一条 SQL');
  const name=await promptDialog({kicker:'保存分析',title:'为这条 SQL 分析命名',label:'分析名称',value:'SQL 分析',placeholder:'例如：每日订单趋势',confirmText:'保存分析'});
  if(!name)return;
  try{
    const saved=await api('/api/questions','POST',{name,query_type:'native',native_sql:sql,database_id:id});
    location.href='/questions/'+saved.id+'/';
  }catch(e){toast(e.message)}
};
$('#schedule').onclick=async()=>{
  try{
    const items=await api('/api/questions');
    if(!items.length) return toast('请先保存一条分析');
    const q=items[0];
    const proposed=await api('/api/ai/propose-schedule','POST',{question_id:q.id,message:$('#question').value||'每天早上九点写入数仓'});
    if(!await confirmDialog({kicker:'AI 调度建议',title:'创建周期性查询任务？',description:proposed.rationale,confirmText:'创建调度',details:[{label:'执行周期',value:proposed.cron},{label:'写入目标',value:proposed.materialize_to},{label:'时区',value:proposed.timezone||'Asia/Shanghai'}]})) return;
    await api('/api/schedules','POST',{name:proposed.name,question_id:q.id,cron:proposed.cron,timezone:proposed.timezone,materialize_to:proposed.materialize_to,strategy:proposed.strategy});
    toast('已创建物化调度，打开数仓页可立即运行');
  }catch(e){toast(e.message)}
};
$('#feishu').onclick=()=>toast('请先在部署配置中填写飞书应用凭据');
boot();
