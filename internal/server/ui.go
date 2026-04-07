package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>Escrow</title>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--orange:#d4843a;--blue:#5b8dd9;--mono:'JetBrains Mono',monospace}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5;font-size:13px;height:100vh;overflow:hidden;display:flex;flex-direction:column}
.hdr{padding:.7rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap;flex-shrink:0}
.hdr h1{font-size:.85rem;letter-spacing:2px}
.hdr h1 span{color:var(--rust)}
.app{flex:1;display:grid;grid-template-columns:260px 1fr;overflow:hidden}
@media(max-width:800px){.app{grid-template-columns:1fr}.sidebar{display:none}}

.sidebar{background:var(--bg2);border-right:1px solid var(--bg3);overflow-y:auto;display:flex;flex-direction:column}
.sidebar-section{padding:.6rem 1rem;font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;display:flex;justify-content:space-between;align-items:center;margin-top:.4rem}
.sidebar-section .add{cursor:pointer;color:var(--cm);background:none;border:none;font-family:var(--mono);font-size:.7rem;padding:0 .2rem}
.sidebar-section .add:hover{color:var(--rust)}
.wf-item{padding:.5rem 1rem;font-size:.7rem;cursor:pointer;display:flex;justify-content:space-between;align-items:center;gap:.4rem;border-left:2px solid transparent;transition:.15s}
.wf-item:hover{background:var(--bg3)}
.wf-item.active{border-left-color:var(--rust);background:var(--bg3);color:var(--cream)}
.wf-name{flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.wf-count{font-size:.55rem;color:var(--cm);background:var(--bg);padding:.05rem .35rem;flex-shrink:0}
.wf-edit{display:none;font-size:.55rem;color:var(--cm);background:none;border:none;cursor:pointer}
.wf-item:hover .wf-edit{display:inline}
.wf-edit:hover{color:var(--cream)}
.stats-bar{padding:.5rem 1rem;border-top:1px solid var(--bg3);font-size:.5rem;color:var(--cm);display:flex;flex-direction:column;gap:.2rem;background:var(--bg2);margin-top:auto}
.stats-bar strong{color:var(--cd);font-weight:700}

.main{overflow-y:auto;padding:1rem 1.5rem}
.toolbar{display:flex;gap:.5rem;margin-bottom:1rem;flex-wrap:wrap;align-items:center}
.toolbar h2{font-size:.85rem;flex:1;color:var(--cream);font-weight:700;letter-spacing:1px;text-transform:uppercase}
.filter-sel{padding:.4rem .5rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.65rem}
.list{display:flex;flex-direction:column;gap:.6rem}
.req{background:var(--bg2);border:1px solid var(--bg3);padding:.9rem 1rem;display:flex;flex-direction:column;gap:.5rem;transition:border-color .15s}
.req:hover{border-color:var(--leather)}
.req.approved{border-left:3px solid var(--green)}
.req.rejected{border-left:3px solid var(--red)}
.req.pending{border-left:3px solid var(--orange)}
.req-hdr{display:flex;justify-content:space-between;align-items:flex-start;gap:.5rem}
.req-title{font-size:.85rem;font-weight:700;color:var(--cream);flex:1}
.req-body{font-size:.7rem;color:var(--cd);line-height:1.5}
.req-meta{display:flex;gap:.5rem;flex-wrap:wrap;align-items:center;font-size:.55rem;color:var(--cm)}
.badge{font-size:.5rem;padding:.12rem .35rem;text-transform:uppercase;letter-spacing:1px;border:1px solid var(--bg3);color:var(--cm);font-weight:700}
.badge.pending{border-color:var(--orange);color:var(--orange)}
.badge.approved{border-color:var(--green);color:var(--green)}
.badge.rejected{border-color:var(--red);color:var(--red)}
.req-decisions{margin-top:.4rem;padding-top:.4rem;border-top:1px dashed var(--bg3);display:flex;flex-direction:column;gap:.3rem}
.dec{font-size:.6rem;display:flex;gap:.5rem;align-items:flex-start}
.dec-icon{flex-shrink:0;font-size:.7rem;line-height:1.2}
.dec-icon.approved{color:var(--green)}
.dec-icon.rejected{color:var(--red)}
.dec-body{flex:1;min-width:0}
.dec-approver{color:var(--cd);font-weight:700}
.dec-comment{color:var(--cm);font-style:italic}
.req-actions{display:flex;gap:.4rem;margin-top:.4rem;flex-wrap:wrap;align-items:center}
.req-actions select{padding:.3rem .4rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.6rem;flex:0 0 140px}
.req-extra{font-size:.55rem;color:var(--cd);margin-top:.4rem;padding-top:.3rem;border-top:1px dashed var(--bg3);display:flex;flex-direction:column;gap:.15rem}
.req-extra-row{display:flex;gap:.4rem}
.req-extra-label{color:var(--cm);text-transform:uppercase;letter-spacing:.5px;min-width:90px}
.req-extra-val{color:var(--cream)}

.btn{font-family:var(--mono);font-size:.6rem;padding:.3rem .55rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:.15s}
.btn:hover{border-color:var(--leather);color:var(--cream)}
.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}
.btn-p:hover{opacity:.85;color:#fff}
.btn-approve{border-color:#1e3a1e;color:var(--green)}
.btn-approve:hover{border-color:var(--green);background:#0f200f}
.btn-reject{border-color:#3a1a1a;color:var(--red)}
.btn-reject:hover{border-color:var(--red);background:#200f0f}
.btn-sm{font-size:.55rem;padding:.2rem .4rem}
.btn-del{color:var(--red);border-color:#3a1a1a}
.btn-del:hover{border-color:var(--red);color:var(--red)}

.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}
.modal-bg.open{display:flex}
.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:520px;max-width:92vw;max-height:90vh;overflow-y:auto}
.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust);letter-spacing:1px}
.fr{margin-bottom:.6rem}
.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}
.fr input,.fr select,.fr textarea{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}
.fr input[type=checkbox]{width:auto}
.fr input:focus,.fr select:focus,.fr textarea:focus{outline:none;border-color:var(--leather)}
.fr-section{margin-top:1rem;padding-top:.8rem;border-top:1px solid var(--bg3)}
.fr-section-label{font-size:.55rem;color:var(--rust);text-transform:uppercase;letter-spacing:1px;margin-bottom:.5rem}
.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}
.acts .btn-del{margin-right:auto}
.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.85rem}
</style>
</head>
<body>

<div class="hdr">
<h1 id="dash-title"><span>&#9670;</span> ESCROW</h1>
<button class="btn btn-p" onclick="openRequestForm()">+ New Request</button>
</div>

<div class="app">

<aside class="sidebar">
<div class="sidebar-section">All Requests <button class="add" onclick="selectWorkflow('')">All</button></div>
<div id="all-link"></div>
<div class="sidebar-section">Workflows <button class="add" onclick="openWorkflowForm()">+</button></div>
<div id="workflows"></div>
<div class="stats-bar" id="stats"></div>
</aside>

<section class="main">
<div class="toolbar">
<h2 id="main-title">All Requests</h2>
<select class="filter-sel" id="status-filter" onchange="render()">
<option value="">All</option>
<option value="pending">Pending</option>
<option value="approved">Approved</option>
<option value="rejected">Rejected</option>
</select>
</div>
<div id="list" class="list"></div>
</section>

</div>

<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()">
<div class="modal" id="mdl"></div>
</div>

<script>
var A='/api';
var workflows=[],requests=[],currentWfId='',workflowExtras={},requestExtras={};
var workflowCustomFields=[],requestCustomFields=[];

function fmtDate(s){
if(!s)return'';
try{
var d=new Date(s);
if(isNaN(d.getTime()))return s;
return d.toLocaleDateString('en-US',{month:'short',day:'numeric',year:'numeric'});
}catch(e){return s}
}

// ─── Loading ──────────────────────────────────────────────────────

async function loadAll(){
try{
var resps=await Promise.all([
fetch(A+'/workflows').then(function(r){return r.json()}),
fetch(A+'/extras/workflows').then(function(r){return r.json()}),
fetch(A+'/extras/requests').then(function(r){return r.json()}),
fetch(A+'/stats').then(function(r){return r.json()})
]);
workflows=resps[0].workflows||[];
workflowExtras=resps[1]||{};
requestExtras=resps[2]||{};
renderSidebar(resps[3]||{});
}catch(e){
console.error('loadAll failed',e);
workflows=[];
}
await loadRequests();
}

async function loadRequests(){
var sf=document.getElementById('status-filter').value;
var qs='';
if(currentWfId)qs+='workflow_id='+encodeURIComponent(currentWfId);
if(sf){if(qs)qs+='&';qs+='status='+encodeURIComponent(sf)}
try{
var r=await fetch(A+'/requests'+(qs?'?'+qs:'')).then(function(r){return r.json()});
requests=r.requests||[];
requests.forEach(function(req){
var x=requestExtras[req.id];
if(!x)return;
Object.keys(x).forEach(function(k){if(req[k]===undefined)req[k]=x[k]});
});
}catch(e){
requests=[];
}
render();
}

function renderSidebar(stats){
var html='';
workflows.forEach(function(wf){
var cls='wf-item'+(wf.id===currentWfId?' active':'');
html+='<div class="'+cls+'" onclick="selectWorkflow(\''+esc(wf.id)+'\')">';
html+='<span class="wf-name">'+esc(wf.name)+'</span>';
html+='<span class="wf-count">'+(wf.request_count||0)+'</span>';
html+='<button class="wf-edit" onclick="event.stopPropagation();openWorkflowForm(\''+esc(wf.id)+'\')">edit</button>';
html+='</div>';
});
if(!workflows.length)html='<div style="padding:.6rem 1rem;font-size:.6rem;color:var(--cm);font-style:italic">No workflows yet</div>';
document.getElementById('workflows').innerHTML=html;

document.getElementById('all-link').innerHTML='<div class="wf-item'+(currentWfId===''?' active':'')+'" onclick="selectWorkflow(\'\')"><span class="wf-name">All Requests</span><span class="wf-count">'+(stats.requests||0)+'</span></div>';

document.getElementById('stats').innerHTML=
'<div><strong>'+(stats.workflows||0)+'</strong> workflows</div>'+
'<div><strong>'+(stats.requests||0)+'</strong> total · <strong>'+(stats.pending||0)+'</strong> pending</div>'+
'<div><strong>'+(stats.approved||0)+'</strong> approved · <strong>'+(stats.rejected||0)+'</strong> rejected</div>';
}

function selectWorkflow(id){
currentWfId=id;
var wf=null;
for(var i=0;i<workflows.length;i++)if(workflows[i].id===id){wf=workflows[i];break}
document.getElementById('main-title').textContent=wf?wf.name:'All Requests';
loadAll();
}

function render(){
if(!requests.length){
var msg=window._emptyMsg||'No requests yet.';
document.getElementById('list').innerHTML='<div class="empty">'+esc(msg)+'</div>';
return;
}
var h='';
requests.forEach(function(r){h+=requestHTML(r)});
document.getElementById('list').innerHTML=h;
}

function requestHTML(req){
var wf=null;
for(var i=0;i<workflows.length;i++)if(workflows[i].id===req.workflow_id){wf=workflows[i];break}

var cls='req '+(req.status||'pending');

var h='<div class="'+cls+'">';
h+='<div class="req-hdr">';
h+='<div class="req-title">'+esc(req.title)+'</div>';
h+='<div style="display:flex;gap:.3rem">';
h+='<button class="btn btn-sm" onclick="openRequestEdit(\''+esc(req.id)+'\')">Edit</button>';
h+='</div>';
h+='</div>';

if(req.body)h+='<div class="req-body">'+esc(req.body)+'</div>';

h+='<div class="req-meta">';
h+='<span class="badge '+esc(req.status)+'">'+esc(req.status)+'</span>';
if(wf)h+='<span>'+esc(wf.name)+'</span>';
if(req.submitter)h+='<span>by '+esc(req.submitter)+'</span>';
h+='<span>'+esc(fmtDate(req.created_at))+'</span>';
if(req.resolved_at)h+='<span>resolved '+esc(fmtDate(req.resolved_at))+'</span>';
h+='</div>';

// Decisions
if(req.decisions&&req.decisions.length){
h+='<div class="req-decisions">';
req.decisions.forEach(function(d){
var icon=d.action==='approved'?'&#10003;':'&#10007;';
h+='<div class="dec">';
h+='<span class="dec-icon '+esc(d.action)+'">'+icon+'</span>';
h+='<div class="dec-body"><span class="dec-approver">'+esc(d.approver)+'</span> '+esc(d.action);
if(d.comment)h+=' <span class="dec-comment">— '+esc(d.comment)+'</span>';
h+='</div></div>';
});
h+='</div>';
}

// Action row for pending requests
if(req.status==='pending'&&wf&&wf.approvers&&wf.approvers.length){
// Filter out approvers who already voted
var voted={};
(req.decisions||[]).forEach(function(d){voted[d.approver]=true});
var available=wf.approvers.filter(function(a){return!voted[a]});
if(available.length){
h+='<div class="req-actions">';
h+='<select id="ap-'+esc(req.id)+'"><option value="">Select approver...</option>';
available.forEach(function(a){h+='<option value="'+esc(a)+'">'+esc(a)+'</option>'});
h+='</select>';
h+='<input type="text" id="cm-'+esc(req.id)+'" placeholder="Comment (optional)" style="flex:1;padding:.3rem .4rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.6rem">';
h+='<button class="btn btn-approve btn-sm" onclick="decide(\''+esc(req.id)+'\',\'approve\')">Approve</button>';
h+='<button class="btn btn-reject btn-sm" onclick="decide(\''+esc(req.id)+'\',\'reject\')">Reject</button>';
h+='</div>';
}
}

// Custom fields
var customRows='';
requestCustomFields.forEach(function(f){
var v=req[f.name];
if(v===undefined||v===null||v==='')return;
customRows+='<div class="req-extra-row">';
customRows+='<span class="req-extra-label">'+esc(f.label)+'</span>';
customRows+='<span class="req-extra-val">'+esc(String(v))+'</span>';
customRows+='</div>';
});
if(customRows)h+='<div class="req-extra">'+customRows+'</div>';

h+='</div>';
return h;
}

// ─── Decisions ────────────────────────────────────────────────────

async function decide(reqId,which){
var apEl=document.getElementById('ap-'+reqId);
var cmEl=document.getElementById('cm-'+reqId);
var approver=apEl?apEl.value:'';
if(!approver){alert('Select an approver');return}
var comment=cmEl?cmEl.value:'';
try{
var r=await fetch(A+'/requests/'+reqId+'/'+which,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({approver:approver,comment:comment})});
if(!r.ok){
var e=await r.json().catch(function(){return{}});
alert(e.error||'Decision failed');
return;
}
}catch(e){alert('Network error: '+e.message);return}
loadAll();
}

// ─── Workflow modal ───────────────────────────────────────────────

function openWorkflowForm(id){
var wf=null;
if(id){for(var i=0;i<workflows.length;i++)if(workflows[i].id===id){wf=workflows[i];break}}
var isEdit=!!wf;
var ext=isEdit?(workflowExtras[id]||{}):{};

var h='<h2>'+(isEdit?'EDIT WORKFLOW':'NEW WORKFLOW')+'</h2>';
h+='<div class="fr"><label>Name *</label><input id="wf-name" value="'+esc(wf?wf.name:'')+'"></div>';
h+='<div class="fr"><label>Description</label><textarea id="wf-description" rows="2">'+esc(wf?(wf.description||''):'')+'</textarea></div>';
h+='<div class="fr"><label>Approvers (one per line)</label><textarea id="wf-approvers" rows="4" placeholder="alice@example.com&#10;bob@example.com">'+esc((wf?(wf.approvers||[]):[]).join('\n'))+'</textarea></div>';
h+='<div class="fr"><label><input type="checkbox" id="wf-require-all"'+(wf&&wf.require_all?' checked':'')+'> Require all approvers (otherwise first approval wins)</label></div>';

if(workflowCustomFields.length){
h+='<div class="fr-section"><div class="fr-section-label">Workflow Details</div>';
workflowCustomFields.forEach(function(f){h+=customFieldHTML('xw',f,ext[f.name])});
h+='</div>';
}

h+='<div class="acts">';
if(isEdit)h+='<button class="btn btn-del" onclick="deleteWorkflow(\''+esc(id)+'\')">Delete</button>';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='<button class="btn btn-p" onclick="saveWorkflow(\''+(id||'')+'\')">'+(isEdit?'Save':'Create')+'</button>';
h+='</div>';

document.getElementById('mdl').innerHTML=h;
document.getElementById('mbg').classList.add('open');
setTimeout(function(){var n=document.getElementById('wf-name');if(n)n.focus()},50);
}

function customFieldHTML(prefix,f,value){
var v=value;
if(v===undefined||v===null)v='';
var h='<div class="fr"><label>'+esc(f.label)+'</label>';
if(f.type==='textarea'){
h+='<textarea id="'+prefix+'-'+f.name+'" rows="2">'+esc(String(v))+'</textarea>';
}else if(f.type==='select'){
h+='<select id="'+prefix+'-'+f.name+'"><option value="">Select...</option>';
(f.options||[]).forEach(function(o){
var sel=String(v)===String(o)?' selected':'';
h+='<option value="'+esc(String(o))+'"'+sel+'>'+esc(String(o))+'</option>';
});
h+='</select>';
}else if(f.type==='number'){
h+='<input type="number" id="'+prefix+'-'+f.name+'" value="'+esc(String(v))+'">';
}else{
h+='<input type="text" id="'+prefix+'-'+f.name+'" value="'+esc(String(v))+'">';
}
h+='</div>';
return h;
}

async function saveWorkflow(id){
var name=document.getElementById('wf-name').value.trim();
if(!name){alert('Name required');return}
var apRaw=document.getElementById('wf-approvers').value;
var ap=apRaw.split('\n').map(function(s){return s.trim()}).filter(function(s){return s});
var body={
name:name,
description:document.getElementById('wf-description').value,
approvers:ap,
require_all:document.getElementById('wf-require-all').checked
};
var extras={};
workflowCustomFields.forEach(function(f){
var el=document.getElementById('xw-'+f.name);
if(!el)return;
extras[f.name]=f.type==='number'?(parseFloat(el.value)||0):el.value.trim();
});

var savedId=id;
try{
if(id){
var r1=await fetch(A+'/workflows/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r1.ok){var e1=await r1.json().catch(function(){return{}});alert(e1.error||'Save failed');return}
}else{
var r2=await fetch(A+'/workflows',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r2.ok){var e2=await r2.json().catch(function(){return{}});alert(e2.error||'Create failed');return}
var created=await r2.json();
savedId=created.id;
}
if(savedId&&Object.keys(extras).length){
await fetch(A+'/extras/workflows/'+savedId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(extras)}).catch(function(){});
}
}catch(e){alert('Network error');return}
closeModal();
loadAll();
}

async function deleteWorkflow(id){
if(!confirm('Delete this workflow and all its requests?'))return;
await fetch(A+'/workflows/'+id,{method:'DELETE'});
if(currentWfId===id)currentWfId='';
closeModal();
loadAll();
}

// ─── Request modal ────────────────────────────────────────────────

function openRequestForm(){
showRequestForm(null);
}

function openRequestEdit(id){
var req=null;
for(var i=0;i<requests.length;i++)if(requests[i].id===id){req=requests[i];break}
if(!req)return;
showRequestForm(req);
}

function showRequestForm(req){
var isEdit=!!req;
var r=req||{title:'',body:'',submitter:'',workflow_id:currentWfId||(workflows[0]?workflows[0].id:'')};
var ext=isEdit?(requestExtras[r.id]||{}):{};

if(!isEdit&&!workflows.length){alert('Create a workflow first');return}

var h='<h2>'+(isEdit?'EDIT REQUEST':'NEW REQUEST')+'</h2>';
if(!isEdit){
h+='<div class="fr"><label>Workflow *</label><select id="rf-workflow">';
workflows.forEach(function(wf){
var sel=wf.id===r.workflow_id?' selected':'';
h+='<option value="'+esc(wf.id)+'"'+sel+'>'+esc(wf.name)+'</option>';
});
h+='</select></div>';
}
h+='<div class="fr"><label>Title *</label><input id="rf-title" value="'+esc(r.title||'')+'"></div>';
h+='<div class="fr"><label>Body</label><textarea id="rf-body" rows="3">'+esc(r.body||'')+'</textarea></div>';
h+='<div class="fr"><label>Submitter</label><input id="rf-submitter" value="'+esc(r.submitter||'')+'"></div>';

if(requestCustomFields.length){
h+='<div class="fr-section"><div class="fr-section-label">Request Details</div>';
requestCustomFields.forEach(function(f){h+=customFieldHTML('xr',f,ext[f.name])});
h+='</div>';
}

h+='<div class="acts">';
if(isEdit)h+='<button class="btn btn-del" onclick="deleteRequest(\''+esc(r.id)+'\')">Delete</button>';
h+='<button class="btn" onclick="closeModal()">Cancel</button>';
h+='<button class="btn btn-p" onclick="saveRequest(\''+(isEdit?esc(r.id):'')+'\')">'+(isEdit?'Save':'Submit')+'</button>';
h+='</div>';

document.getElementById('mdl').innerHTML=h;
document.getElementById('mbg').classList.add('open');
}

async function saveRequest(id){
var title=document.getElementById('rf-title').value.trim();
if(!title){alert('Title required');return}
var body={
title:title,
body:document.getElementById('rf-body').value,
submitter:document.getElementById('rf-submitter').value.trim()
};
if(!id){
var wfEl=document.getElementById('rf-workflow');
body.workflow_id=wfEl?wfEl.value:'';
if(!body.workflow_id){alert('Workflow required');return}
}
var extras={};
requestCustomFields.forEach(function(f){
var el=document.getElementById('xr-'+f.name);
if(!el)return;
extras[f.name]=f.type==='number'?(parseFloat(el.value)||0):el.value.trim();
});

var savedId=id;
try{
if(id){
var r1=await fetch(A+'/requests/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r1.ok){var e1=await r1.json().catch(function(){return{}});alert(e1.error||'Save failed');return}
}else{
var r2=await fetch(A+'/requests',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(!r2.ok){var e2=await r2.json().catch(function(){return{}});alert(e2.error||'Submit failed');return}
var created=await r2.json();
savedId=created.id;
}
if(savedId&&Object.keys(extras).length){
await fetch(A+'/extras/requests/'+savedId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(extras)}).catch(function(){});
}
}catch(e){alert('Network error');return}
closeModal();
loadAll();
}

async function deleteRequest(id){
if(!confirm('Delete this request?'))return;
await fetch(A+'/requests/'+id,{method:'DELETE'});
closeModal();
loadAll();
}

function closeModal(){
document.getElementById('mbg').classList.remove('open');
}

function esc(s){
if(s===undefined||s===null)return'';
var d=document.createElement('div');
d.textContent=String(s);
return d.innerHTML;
}

document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal()});

// ─── Personalization ──────────────────────────────────────────────

(function loadPersonalization(){
fetch('/api/config').then(function(r){return r.json()}).then(function(cfg){
if(!cfg||typeof cfg!=='object')return;

if(cfg.dashboard_title){
var h1=document.getElementById('dash-title');
if(h1)h1.innerHTML='<span>&#9670;</span> '+esc(cfg.dashboard_title);
document.title=cfg.dashboard_title;
}

if(cfg.empty_state_message)window._emptyMsg=cfg.empty_state_message;

if(Array.isArray(cfg.workflow_custom_fields)){
workflowCustomFields=cfg.workflow_custom_fields.filter(function(f){return f&&f.name&&f.label});
}
if(Array.isArray(cfg.request_custom_fields)){
requestCustomFields=cfg.request_custom_fields.filter(function(f){return f&&f.name&&f.label});
}
}).catch(function(){
}).finally(function(){
loadAll();
});
})();
</script>
</body>
</html>`
