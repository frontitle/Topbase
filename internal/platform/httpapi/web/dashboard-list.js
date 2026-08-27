async function load(){
  const [raw, projects]=await Promise.all([api('/api/dashboards'),api('/api/collections')]);
  const boards=Array.isArray(raw)?raw:[];
  const projectName=id=>{const p=projects.find(x=>x.id===id);return p?p.name:'我的分析'};
  $('#list').innerHTML=boards.map(board=>cardHTML({
    href:'/dashboard/'+board.id+'/',
    title:board.name,
    meta:projectName(board.collection_id)+' · '+(board.cards||[]).length+' 张卡片 · '+(board.filters||[]).length+' 个筛选'
  })).join('')||emptyHTML({icon:'☷',title:'还没有仪表盘',body:'创建后会直接进入空白编辑器，从左侧选择分析即可。',href:'#',cta:'新建仪表盘'});
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
