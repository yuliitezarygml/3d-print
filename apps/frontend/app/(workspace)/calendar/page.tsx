"use client";

import { useQuery } from "@tanstack/react-query";
import { CalendarDays, Clock3, Printer } from "lucide-react";
import { api } from "@/lib/api";

type CalendarJob={id:string;status:string;printerName:string;orderNumber:string;modelName:string;start:string;end:string};
const statusLabels:Record<string,string>={QUEUED:"В очереди",READY:"Готов к запуску",PRINTING:"Печатается",PAUSED:"Пауза",SUCCESS:"Завершено",FAILED:"Ошибка",CANCELLED:"Отменено"};

export default function CalendarPage(){
  const start=new Date();start.setDate(start.getDate()-3);const end=new Date();end.setDate(end.getDate()+28);
  const jobs=useQuery({queryKey:["calendar"],queryFn:()=>api<CalendarJob[]>(`/api/calendar?from=${encodeURIComponent(start.toISOString())}&to=${encodeURIComponent(end.toISOString())}`)});
  const grouped=(jobs.data||[]).reduce<Record<string,CalendarJob[]>>((result,item)=>{const key=new Date(item.start).toLocaleDateString("ru-MD",{weekday:"long",day:"numeric",month:"long"});(result[key]||=[]).push(item);return result},{});
  return <><div className="page-title"><div><p className="eyebrow">ПРОИЗВОДСТВО</p><h1>Календарь печати</h1><p>Расписание принтеров на четыре недели вперёд.</p></div><span className="hero-icon"><CalendarDays/></span></div><section className="calendar-board">{jobs.isPending?<div className="empty-state">Загружаем расписание…</div>:Object.keys(grouped).length?Object.entries(grouped).map(([day,items])=><div className="calendar-day" key={day}><header><strong>{day}</strong><span>{items.length} заданий</span></header><div>{items.map(item=><article key={item.id} className={`calendar-job ${item.status.toLowerCase()}`}><i/><div><strong>{item.modelName||item.orderNumber||"Задание печати"}</strong><span><Printer size={14}/>{item.printerName}</span></div><div className="calendar-time"><strong>{new Date(item.start).toLocaleTimeString("ru-MD",{hour:"2-digit",minute:"2-digit"})}</strong><span><Clock3 size={13}/>{Math.max(1,Math.round((+new Date(item.end)-+new Date(item.start))/60000))} мин</span></div><em>{statusLabels[item.status]||item.status}</em></article>)}</div></div>):<div className="empty-state"><CalendarDays/><h2>Расписание свободно</h2><p>Укажите время старта при создании задания печати.</p></div>}</section></>;
}
