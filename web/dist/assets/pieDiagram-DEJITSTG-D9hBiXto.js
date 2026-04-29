import{aF as S,ax as R,b6 as K,B as Q,z as tt,C as et,D as at,Y as rt,X as nt,_ as p,G as z,E as it,a9 as st,ac as ot,ag as lt,H as ct,a3 as ut,aa as pt}from"./index-By6rNCYx.js";import{p as gt}from"./chunk-4BX2VUAB-CtDgT7IM.js";import{p as dt}from"./wardley-RL74JXVD-qGKzXyDB.js";import{d as _}from"./arc-9-UNyVLa.js";import{o as ft}from"./ordinal-DxaMzblD.js";import"./semi-ui-C9x-3bX2.js";import"./react-core-DBdLxtWm.js";import"./i18n-D6tBGcxi.js";import"./tools-z_LLUnnf.js";import"./react-components-CLr3v-HB.js";import"./min-B9W2RHts.js";import"./_baseUniq-CozEelu4.js";import"./init-Gi6I4Gst.js";function ht(t,a){return a<t?-1:a>t?1:a>=t?0:NaN}function mt(t){return t}function vt(){var t=mt,a=ht,f=null,y=S(0),s=S(R),g=S(0);function o(e){var n,l=(e=K(e)).length,d,h,v=0,c=new Array(l),i=new Array(l),x=+y.apply(this,arguments),w=Math.min(R,Math.max(-R,s.apply(this,arguments)-x)),m,D=Math.min(Math.abs(w)/l,g.apply(this,arguments)),$=D*(w<0?-1:1),u;for(n=0;n<l;++n)(u=i[c[n]=n]=+t(e[n],n,e))>0&&(v+=u);for(a!=null?c.sort(function(A,C){return a(i[A],i[C])}):f!=null&&c.sort(function(A,C){return f(e[A],e[C])}),n=0,h=v?(w-l*$)/v:0;n<l;++n,x=m)d=c[n],u=i[d],m=x+(u>0?u*h:0)+$,i[d]={data:e[d],index:n,value:u,startAngle:x,endAngle:m,padAngle:D};return i}return o.value=function(e){return arguments.length?(t=typeof e=="function"?e:S(+e),o):t},o.sortValues=function(e){return arguments.length?(a=e,f=null,o):a},o.sort=function(e){return arguments.length?(f=e,a=null,o):f},o.startAngle=function(e){return arguments.length?(y=typeof e=="function"?e:S(+e),o):y},o.endAngle=function(e){return arguments.length?(s=typeof e=="function"?e:S(+e),o):s},o.padAngle=function(e){return arguments.length?(g=typeof e=="function"?e:S(+e),o):g},o}var xt=pt.pie,F={sections:new Map,showData:!1},T=F.sections,W=F.showData,St=structuredClone(xt),yt=p(()=>structuredClone(St),"getConfig"),wt=p(()=>{T=new Map,W=F.showData,ut()},"clear"),At=p(({label:t,value:a})=>{if(a<0)throw new Error(`"${t}" has invalid value: ${a}. Negative values are not allowed in pie charts. All slice values must be >= 0.`);T.has(t)||(T.set(t,a),z.debug(`added new section: ${t}, with value: ${a}`))},"addSection"),Ct=p(()=>T,"getSections"),Dt=p(t=>{W=t},"setShowData"),$t=p(()=>W,"getShowData"),V={getConfig:yt,clear:wt,setDiagramTitle:nt,getDiagramTitle:rt,setAccTitle:at,getAccTitle:et,setAccDescription:tt,getAccDescription:Q,addSection:At,getSections:Ct,setShowData:Dt,getShowData:$t},Tt=p((t,a)=>{gt(t,a),a.setShowData(t.showData),t.sections.map(a.addSection)},"populateDb"),Et={parse:p(async t=>{const a=await dt("pie",t);z.debug(a),Tt(a,V)},"parse")},bt=p(t=>`
  .pieCircle{
    stroke: ${t.pieStrokeColor};
    stroke-width : ${t.pieStrokeWidth};
    opacity : ${t.pieOpacity};
  }
  .pieOuterCircle{
    stroke: ${t.pieOuterStrokeColor};
    stroke-width: ${t.pieOuterStrokeWidth};
    fill: none;
  }
  .pieTitleText {
    text-anchor: middle;
    font-size: ${t.pieTitleTextSize};
    fill: ${t.pieTitleTextColor};
    font-family: ${t.fontFamily};
  }
  .slice {
    font-family: ${t.fontFamily};
    fill: ${t.pieSectionTextColor};
    font-size:${t.pieSectionTextSize};
    // fill: white;
  }
  .legend text {
    fill: ${t.pieLegendTextColor};
    font-family: ${t.fontFamily};
    font-size: ${t.pieLegendTextSize};
  }
`,"getStyles"),kt=bt,Mt=p(t=>{const a=[...t.values()].reduce((s,g)=>s+g,0),f=[...t.entries()].map(([s,g])=>({label:s,value:g})).filter(s=>s.value/a*100>=1);return vt().value(s=>s.value).sort(null)(f)},"createPieArcs"),Rt=p((t,a,f,y)=>{var P;z.debug(`rendering pie chart
`+t);const s=y.db,g=it(),o=st(s.getConfig(),g.pie),e=40,n=18,l=4,d=450,h=d,v=ot(a),c=v.append("g");c.attr("transform","translate("+h/2+","+d/2+")");const{themeVariables:i}=g;let[x]=lt(i.pieOuterStrokeWidth);x??(x=2);const w=o.textPosition,m=Math.min(h,d)/2-e,D=_().innerRadius(0).outerRadius(m),$=_().innerRadius(m*w).outerRadius(m*w);c.append("circle").attr("cx",0).attr("cy",0).attr("r",m+x/2).attr("class","pieOuterCircle");const u=s.getSections(),A=Mt(u),C=[i.pie1,i.pie2,i.pie3,i.pie4,i.pie5,i.pie6,i.pie7,i.pie8,i.pie9,i.pie10,i.pie11,i.pie12];let E=0;u.forEach(r=>{E+=r});const G=A.filter(r=>(r.data.value/E*100).toFixed(0)!=="0"),b=ft(C).domain([...u.keys()]);c.selectAll("mySlices").data(G).enter().append("path").attr("d",D).attr("fill",r=>b(r.data.label)).attr("class","pieCircle"),c.selectAll("mySlices").data(G).enter().append("text").text(r=>(r.data.value/E*100).toFixed(0)+"%").attr("transform",r=>"translate("+$.centroid(r)+")").style("text-anchor","middle").attr("class","slice");const U=c.append("text").text(s.getDiagramTitle()).attr("x",0).attr("y",-400/2).attr("class","pieTitleText"),B=[...u.entries()].map(([r,M])=>({label:r,value:M})),k=c.selectAll(".legend").data(B).enter().append("g").attr("class","legend").attr("transform",(r,M)=>{const I=n+l,Z=I*B.length/2,q=12*n,J=M*I-Z;return"translate("+q+","+J+")"});k.append("rect").attr("width",n).attr("height",n).style("fill",r=>b(r.label)).style("stroke",r=>b(r.label)),k.append("text").attr("x",n+l).attr("y",n-l).text(r=>s.getShowData()?`${r.label} [${r.value}]`:r.label);const X=Math.max(...k.selectAll("text").nodes().map(r=>(r==null?void 0:r.getBoundingClientRect().width)??0)),j=h+e+n+l+X,L=((P=U.node())==null?void 0:P.getBoundingClientRect().width)??0,H=h/2-L/2,Y=h/2+L/2,N=Math.min(0,H),O=Math.max(j,Y)-N;v.attr("viewBox",`${N} 0 ${O} ${d}`),ct(v,d,O,o.useMaxWidth)},"draw"),zt={draw:Rt},Ht={parser:Et,db:V,renderer:zt,styles:kt};export{Ht as diagram};
