let proposal=null;

async function load(){
  const [tables, schedules, questions]=await Promise.all([
    api('/api/warehouse/tables'), api('/api/schedules'), api('/api/questions')
  ]);
  $('#tables').innerHTML=tables.map(t=>cardHTML({title:t.schema+'.'+t.name,meta:'数仓 · '+(t.row_count||0)+' 行 · '+(t.last_status||'尚未运行')+(t.watermark?' · watermark '+t.watermark:'')})).join('')||emptyHTML({icon:'▣',title:'还没有物化表',body:'用下面的表单把一条分析升级成数仓表。',href:'/questions/',cta:'先选一条分析'});
  $('#schedules').innerHTML=schedules.map(s=>cardHTML({title:s.name,meta:(s.strategy||'replace')+(s.watermark_field?' / '+s.watermark_field:'')+' · '+s.cron+' → '+s.materialize_to,action:'<button class="secondary" data-run="'+esc(s.id)+'" type="button">立即运行</button>'})).join('')||emptyHTML({icon:'▣',title:'还没有调度',body:'先让 AI 提案，确认后再创建。'});
  $('#questions').innerHTML=questions.map(q=>'<option value="'+q.id+'">'+esc(q.name)+'</option>').join('');
  $$('[data-run]').forEach(b=>b.onclick=async()=>{
    try{const run=await api('/api/schedules/'+b.dataset.run+'/run','POST',{});toast('已写入 '+run.row_count+' 行');load()}catch(e){toast(e.message)}
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
  if(!proposal) return toast('请先生成提案');
  try{
    await api('/api/schedules','POST',{
      name:proposal.name, question_id:$('#questions').value, cron:proposal.cron,
      timezone:proposal.timezone, materialize_to:proposal.materialize_to, strategy:proposal.strategy,
      watermark_field:proposal.watermark_field
    });
    toast('调度已创建');
    proposal=null;
    load();
  }catch(e){toast(e.message)}
};

load().catch(e=>toast(e.message));
