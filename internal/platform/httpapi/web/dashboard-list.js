async function load(){
  const [raw, projects]=await Promise.all([api('/api/dashboards'),api('/api/collections')]);
  const boards=Array.isArray(raw)?raw:[];
  const projectName=id=>{const p=projects.find(x=>x.id===id);return p?p.name:window.TopbaseLocale.t('我的分析')};
  $('#list').innerHTML=boards.map(board=>cardHTML({
    href:'/dashboard/'+board.id+'/',
    title:(dashboardShareStatus(board) === 'published' ? '◉ ' : '')+board.name,
    meta:(dashboardShareStatus(board) === 'expired' ? window.TopbaseLocale.t('分享已到期')+' · ' : '')+projectName(board.collection_id)+' · '+window.TopbaseLocale.t('{cards} 张卡片 · {filters} 个筛选', {cards:(board.cards||[]).length, filters:(board.filters||[]).length})
  })).join('')||emptyHTML({icon:'☷',title:'还没有仪表盘',body:'新建后，点击「插入」即可添加分析或小组件。',href:'#',cta:'新建仪表盘'});
  document.querySelectorAll('#list .db-card b, #list .db-card small').forEach(node=>{node.translate=false;});
  const emptyCTA=document.querySelector('#list .empty a');
  if(emptyCTA)emptyCTA.onclick=event=>{event.preventDefault();createDashboard()};
}

async function createDashboard(){
  const button=$('#show-create');
  if(button.disabled)return;
  button.disabled=true;
  button.textContent='正在创建…';
  try{
    const board=await api('/api/dashboards','POST',{});
    if(!board||!board.id)throw Error('服务器没有返回仪表盘 ID');
    location.href='/dashboard/'+board.id+'/';
  }catch(error){
    button.disabled=false;
    button.textContent='新建仪表盘';
    toast(error.message);
  }
}

$('#show-create').onclick=createDashboard;
load().catch(error=>toast(error.message));
