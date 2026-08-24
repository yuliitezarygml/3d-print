"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Box, Check, Copy, ExternalLink, Plus } from "lucide-react";
import { api, money } from "@/lib/api";
import { ErrorNote, Modal, PageHeader, Status } from "@/components/ui";

type OrderModel={id:string;name:string;originalFilename:string;previewUrl?:string};
type Order={id:string;number:string;trackingCode:string;status:string;statusLabel:string;sellingPrice:number;paidAmount:number;balanceDue:number;deadline?:string;customerId?:string;customerName?:string;models:OrderModel[]};
type Customer={id:string;name:string};
type Model={id:string;name:string;originalFilename:string;format:string;customerId?:string;customerName?:string};
type CreatedOrder={id:string;number:string;trackingCode:string};

const statusOptions=[["NEW","Заказ принят"],["CONFIRMED","Подтверждён"],["WAITING","Ожидает"],["READY_TO_PRINT","Готов к печати"],["PRINTING","Печатается"],["POST_PROCESSING","Постобработка"],["READY","Готов к выдаче"],["COMPLETED","Выдан"],["CANCELLED","Отменён"]];

export default function Orders(){
  const [open,setOpen]=useState(false);
  const [created,setCreated]=useState<CreatedOrder|null>(null);
  const queryClient=useQueryClient();
  const orders=useQuery({queryKey:["orders"],queryFn:()=>api<Order[]>("/api/orders")});
  const customers=useQuery({queryKey:["customers"],queryFn:()=>api<Customer[]>("/api/customers")});
  const models=useQuery({queryKey:["models"],queryFn:()=>api<Model[]>("/api/models")});
  const add=useMutation({mutationFn:(value:Record<string,unknown>)=>api<CreatedOrder>("/api/orders",{method:"POST",body:JSON.stringify(value)}),onSuccess:data=>{queryClient.invalidateQueries({queryKey:["orders"]});queryClient.invalidateQueries({queryKey:["models"]});setOpen(false);setCreated(data)}});
  const status=useMutation({mutationFn:({id,value}:{id:string;value:string})=>api(`/api/orders/${id}/status`,{method:"PATCH",body:JSON.stringify({status:value})}),onSuccess:()=>queryClient.invalidateQueries({queryKey:["orders"]})});
  return <>
    <PageHeader eyebrow="ПРОДАЖИ" title="Заказы" description="У каждого заказа есть безопасный код, страница отслеживания и прикреплённые модели." actions={<button className="button primary" onClick={()=>setOpen(true)}><Plus size={17}/>Новый заказ</button>}/>
    {created&&<div className="tracking-created"><span className="tracking-created-icon"><Check/></span><div><strong>{created.number} создан</strong><p>Передайте клиенту код <b>{created.trackingCode}</b> — он работает на странице и в Telegram‑боте.</p></div><button className="button" onClick={()=>navigator.clipboard.writeText(created.trackingCode)}><Copy size={15}/>Копировать код</button><Link className="button primary" href={`/track/${created.trackingCode}`} target="_blank">Открыть <ExternalLink size={14}/></Link></div>}
    <div className="order-grid">{orders.data?.map(order=><article className="order-card" key={order.id}>
      <div className="order-card-head"><div><p className="eyebrow">{order.number}</p><h2>{order.customerName||"Без клиента"}</h2></div><Status value={order.status}/></div>
      <div className="tracking-code"><span>Код клиента</span><strong>{order.trackingCode}</strong><button className="icon-button" title="Копировать код" onClick={()=>navigator.clipboard.writeText(order.trackingCode)}><Copy size={15}/></button></div>
      <div className="order-models">{order.models.length?order.models.map(model=><span key={model.id}><Box size={14}/>{model.name}</span>):<small>Модели пока не прикреплены</small>}</div>
      <div className="order-money"><div><span>Стоимость</span><strong>{money(order.sellingPrice)}</strong></div><div><span>Оплачено</span><strong>{money(order.paidAmount)}</strong></div><div><span>Остаток</span><strong>{money(order.balanceDue)}</strong></div></div>
      <div className="order-card-actions"><select aria-label={`Статус ${order.number}`} value={order.status} onChange={event=>status.mutate({id:order.id,value:event.target.value})}>{statusOptions.map(([value,label])=><option value={value} key={value}>{label}</option>)}</select><Link className="button" href={`/track/${order.trackingCode}`} target="_blank">Страница клиента <ExternalLink size={14}/></Link></div>
    </article>)}</div>
    {open&&<Modal title="Новый заказ" size="wide" onClose={()=>setOpen(false)}><OrderForm customers={customers.data??[]} models={models.data??[]} busy={add.isPending} error={add.error} onCancel={()=>setOpen(false)} onSubmit={value=>add.mutate(value)}/></Modal>}
  </>;
}

function OrderForm({customers,models,busy,error,onCancel,onSubmit}:{customers:Customer[];models:Model[];busy:boolean;error:unknown;onCancel:()=>void;onSubmit:(value:Record<string,unknown>)=>void}){
  const [customerId,setCustomerId]=useState("");
  const [selected,setSelected]=useState<string[]>([]);
  const available=models.filter(model=>customerId?(!model.customerId||model.customerId===customerId):!model.customerId);
  function submit(event:React.FormEvent<HTMLFormElement>){event.preventDefault();const data=new FormData(event.currentTarget);onSubmit({customerId:customerId||null,deadline:data.get("deadline")?new Date(String(data.get("deadline"))).toISOString():null,sellingPrice:Number(data.get("price")),paidAmount:Number(data.get("paid")),notes:data.get("notes"),modelIds:selected})}
  return <form className="form-grid" onSubmit={submit}>
    <label className="field full">Клиент<select value={customerId} onChange={event=>{setCustomerId(event.target.value);setSelected([])}}><option value="">Без клиента</option>{customers.map(customer=><option value={customer.id} key={customer.id}>{customer.name}</option>)}</select></label>
    <div className="field full"><span>Модели клиента</span><div className="model-picker">{available.length?available.map(model=><label key={model.id} className={selected.includes(model.id)?"selected":""}><input type="checkbox" checked={selected.includes(model.id)} onChange={event=>setSelected(current=>event.target.checked?[...current,model.id]:current.filter(id=>id!==model.id))}/><Box size={18}/><span><strong>{model.name}</strong><small>{model.originalFilename} · {model.customerName||"Общая библиотека"}</small></span></label>):<p>Сначала загрузите модель в библиотеку этого клиента.</p>}</div></div>
    <label className="field">Срок<input name="deadline" type="datetime-local"/></label><label className="field">Стоимость, MDL<input name="price" type="number" min="0" step=".01" defaultValue="0"/></label><label className="field">Оплачено, MDL<input name="paid" type="number" min="0" step=".01" defaultValue="0"/></label><label className="field full">Комментарий<textarea name="notes" rows={3}/></label>
    {Boolean(error)&&<div className="field full"><ErrorNote error={error}/></div>}<div className="form-actions"><button type="button" className="button" onClick={onCancel}>Отмена</button><button className="button primary" disabled={busy}>{busy?"Создаём…":"Создать заказ и код"}</button></div>
  </form>;
}
