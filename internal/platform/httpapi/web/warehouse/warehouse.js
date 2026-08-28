let proposal=null;

async function load(){
  const [tables, schedules, questions]=await Promise.all([
    api('/api/warehouse/tables'), api('/api/schedules'), api('/api/questions')
  ]);
  $('#tables').innerHTML=tables.map(t=>cardHTML({title:t.schema+'.'+t.name,meta:'本地沉淀 · '+(t.row_count||0)+' 行 · '+(t.last_status||'尚未更新')+(t.watermark?' · 增量位置 '+t.watermark:'')})).join('')||emptyHTML({icon:'▣',title:'还没有沉淀数据',body:'选择一条已保存分析，为它创建周期更新计划。',href:'/questions/',cta:'先选择分析'});
  $('#schedules').innerHTML=schedules.map(s=>cardHTML({title:s.name,meta:(s.strategy||'replace')+(s.watermark_field?' / '+s.watermark_field:'')+' · '+s.cron+' → '+s.materialize_to,action:'<button class="secondary" data-run="'+esc(s.id)+'" type="button">立即更新</button>'})).join('')||emptyHTML({icon:'▣',title:'还没有更新计划',body:'先让 AI 生成计划，确认后创建。'});
  $('#questions').innerHTML=questions.map(q=>'<option value="'+q.id+'">'+esc(q.name)+'</option>').join('');
  $$('[data-run]').forEach(b=>b.onclick=async()=>{
    try{const run=await api('/api/schedules/'+b.dataset.run+'/run','POST',{});toast('已沉淀 '+run.row_count+' 行');load()}catch(e){toast(e.message)}
  });
}

$('#propose').onclick=async()=>{
  try{
    proposal=await api('/api/ai/propose-schedule','POST',{question_id:$('#questions').value,message:$('#message').value});
    $('#proposal').textContent=proposal.rationale+'\n'+proposal.cron+' → '+proposal.materialize_to+'（'+proposal.strategy+(proposal.watermark_field?' / '+proposal.watermark_field:'')+'，需确认）';
  }catch(e){toast(e.message)}
};

$('#form').onsubmit=async ev=>{
  ev.preventDefault();
  if(!proposal) return toast('请先生成更新计划');
  try{
    await api('/api/schedules','POST',{
      name:proposal.name, question_id:$('#questions').value, cron:proposal.cron,
      timezone:proposal.timezone, materialize_to:proposal.materialize_to, strategy:proposal.strategy,
      watermark_field:proposal.watermark_field
    });
    toast('更新计划已创建');
    proposal=null;
    load();
  }catch(e){toast(e.message)}
};

load().catch(e=>toast(e.message));
