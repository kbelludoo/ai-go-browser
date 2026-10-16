package benchmark

// The modern Playwright observation reads the selected document's composed
// tree. It projects slots and visits open roots once, without aggregating frames.
const modernCompactDOM = `(() => {
 const elements=[],texts=[],seen=new WeakSet();let visited=0;
 const walk=(node,depth=0)=>{
  if(depth>512||++visited>50000)throw new Error('composed observation exceeds limits');
  if(seen.has(node))return;seen.add(node);
  if(node.nodeType===Node.TEXT_NODE){const text=(node.textContent||'').replace(/\s+/g,' ').trim();if(text)texts.push(text);return}
  if(node.nodeType!==Node.ELEMENT_NODE)return;
  if(['SCRIPT','STYLE','NOSCRIPT','TEMPLATE'].includes(node.tagName))return;
  const style=getComputedStyle(node);if(node.hidden||style.display==='none'||style.visibility==='hidden')return;
  elements.push(node);
  const children=node.tagName==='SLOT'?(node.assignedNodes({flatten:true}).length?node.assignedNodes({flatten:true}):node.childNodes):node.shadowRoot?node.shadowRoot.childNodes:node.childNodes;
  for(const child of children)walk(child,depth+1);
 };
 if(document.body)walk(document.body);
 const short=text=>Array.from(text||'').slice(0,5000).join('');
 return {url:location.href,title:document.title,status:performance.getEntriesByType('navigation')[0]?.responseStatus||0,
 text:short(texts.join(' ')),
 headings:elements.filter(e=>/^H[1-6]$/.test(e.tagName)).slice(0,100).map(e=>({level:Number(e.tagName.slice(1)),text:e.innerText})),
 links:elements.filter(e=>e.matches('a[href]')).slice(0,200).map(e=>({label:e.innerText,url:e.href})),
 buttons:elements.filter(e=>e.matches('button,input[type=button],input[type=submit]')).slice(0,200).map(e=>({id:e.id,label:e.innerText||e.value,type:e.type})),
 forms:elements.filter(e=>e.tagName==='FORM').slice(0,200).map(e=>({action:e.action,method:e.method,fields:Array.from(e.elements).filter(f=>f.name).slice(0,200).map(f=>({name:f.name,type:f.type,value:f.value,label:f.labels?.[0]?.innerText||''}))})),
 document_bytes:new TextEncoder().encode(document.documentElement.outerHTML).length};
})()`
