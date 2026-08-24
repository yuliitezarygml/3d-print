"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Box, CalendarClock, Check, Download, FileDown, PackageCheck, Wallet } from "lucide-react";
import { API_URL, money } from "@/lib/api";

type TrackedModel={id:string;name:string;originalFilename:string;format:string;fileSizeBytes:number;previewUrl?:string;downloadUrl:string};
type TrackedOrder={number:string;trackingCode:string;status:string;statusLabel:string;sellingPrice:number;paidAmount:number;balanceDue:number;currency:string;customerName?:string;deadline?:string;createdAt:string;notes:string;progress:number;models:TrackedModel[]};

const timeline=[{key:"NEW",label:"Принят"},{key:"CONFIRMED",label:"Подтверждён"},{key:"READY_TO_PRINT",label:"Подготовлен"},{key:"PRINTING",label:"Печатается"},{key:"POST_PROCESSING",label:"Обработка"},{key:"READY",label:"Готов"},{key:"COMPLETED",label:"Получен"}];

export default function TrackOrderPage(){
  const params=useParams<{code:string}>();
  const order=useQuery({queryKey:["public-order",params.code],queryFn:async()=>{const response=await fetch(`${API_URL}/api/public/track/${encodeURIComponent(params.code)}`,{cache:"no-store"});if(!response.ok)throw new Error("Заказ с таким кодом не найден");return response.json() as Promise<TrackedOrder>}});
  if(order.isPending)return <main className="tracking-page"><div className="tracking-loader"><span className="brand-mark"><Box/></span><p>Ищем ваш заказ…</p></div></main>;
  if(order.error||!order.data)return <main className="tracking-page"><div className="tracking-error"><span className="brand-mark"><Box/></span><h1>Заказ не найден</h1><p>Проверьте 10-значный код или запросите его у мастерской.</p></div></main>;
  const data=order.data;const currentIndex=timeline.findIndex(item=>item.key===data.status);const cancelled=data.status==="CANCELLED";
  return <main className="tracking-page"><div className="tracking-shell">
    <header className="tracking-top"><div className="brand"><span className="brand-mark"><Box size={20}/></span>PrintForge</div><span>Отслеживание заказа</span></header>
    <section className="tracking-hero"><div><p className="eyebrow">ЗАКАЗ {data.number}</p><h1>{data.customerName?`${data.customerName}, ваш заказ` : "Ваш заказ"}: <em>{data.statusLabel.toLowerCase()}</em></h1><p>Код отслеживания <strong>{data.trackingCode}</strong></p></div><div className="tracking-percent"><strong>{data.progress}%</strong><span>готовность</span></div></section>
    <div className="tracking-progress"><i style={{width:`${data.progress}%`}}/></div>
    <section className={`tracking-timeline ${cancelled?"cancelled":""}`}>{cancelled?<div className="tracking-cancelled">Заказ отменён. Свяжитесь с мастерской для уточнения.</div>:timeline.map((item,index)=><div className={index<=currentIndex?"done":""} key={item.key}><span>{index<currentIndex?<Check size={14}/>:index+1}</span><small>{item.label}</small></div>)}</section>
    <div className="tracking-grid"><section className="tracking-card"><div className="tracking-card-title"><PackageCheck/><div><p className="eyebrow">ВАШИ МОДЕЛИ</p><h2>Файлы заказа</h2></div></div><div className="public-model-grid">{data.models.length?data.models.map(model=><article key={model.id}><div className="public-model-cover" style={model.previewUrl?{backgroundImage:`url("${API_URL}${model.previewUrl}")`}:undefined}>{!model.previewUrl&&<Box size={34}/>}</div><div><strong>{model.name}</strong><small>{model.format} · {(model.fileSizeBytes/1024/1024).toFixed(2)} МБ</small><a className="button" href={`${API_URL}${model.downloadUrl}`} download={model.originalFilename}><Download size={14}/>Скачать модель</a></div></article>):<p className="muted">Файлы появятся после прикрепления моделью мастерской.</p>}</div></section>
      <aside className="tracking-card tracking-finance"><div className="tracking-card-title"><Wallet/><div><p className="eyebrow">СТОИМОСТЬ</p><h2>Оплата заказа</h2></div></div><div className="finance-total"><span>Полная стоимость</span><strong>{money(data.sellingPrice,data.currency)}</strong></div><div className="finance-line"><span>Оплачено</span><strong>{money(data.paidAmount,data.currency)}</strong></div><div className="finance-line due"><span>Осталось оплатить</span><strong>{money(data.balanceDue,data.currency)}</strong></div><a className="button primary receipt-download" href={`${API_URL}/api/public/track/${encodeURIComponent(data.trackingCode)}/receipt.pdf`} download={`receipt-${data.number}.pdf`}><FileDown size={15}/>Скачать PDF-квитанцию</a>{data.deadline&&<div className="tracking-deadline"><CalendarClock size={17}/><span>Плановый срок<strong>{new Date(data.deadline).toLocaleString("ru-MD",{dateStyle:"medium",timeStyle:"short"})}</strong></span></div>}</aside>
    </div>
    <footer className="tracking-footer">Данные обновляются автоматически · код можно отправить Telegram‑боту мастерской</footer>
  </div></main>;
}
