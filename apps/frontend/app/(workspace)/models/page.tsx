"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Camera, Download, FileUp, Layers3, User } from "lucide-react";
import { api, downloadAuthenticated } from "@/lib/api";
import { ErrorNote, PageHeader } from "@/components/ui";
import { ModelViewer } from "@/components/model-viewer";

type Model={id:string;name:string;originalFilename:string;format:string;fileSizeBytes:number;dimensionsXmm?:number;dimensionsYmm?:number;dimensionsZmm?:number;volumeCm3?:number;triangleCount?:number;version:number;customerId?:string;customerName?:string;previewUrl?:string;fileUrl:string};
type Customer={id:string;name:string};

export default function Models(){
  const queryClient=useQueryClient();
  const [selected,setSelected]=useState<Model|null>(null);
  const [customerId,setCustomerId]=useState("all");
  const models=useQuery({queryKey:["models"],queryFn:()=>api<Model[]>("/api/models")});
  const customers=useQuery({queryKey:["customers"],queryFn:()=>api<Customer[]>("/api/customers")});
  const visible=useMemo(()=>models.data?.filter(model=>customerId==="all"||customerId==="shared"?!model.customerId:model.customerId===customerId)??[],[models.data,customerId]);
  const active=selected&&visible.some(model=>model.id===selected.id)?selected:visible[0];
  const upload=useMutation({mutationFn:({file,owner}:{file:File;owner:string})=>{const body=new FormData();body.set("file",file);if(owner!=="all"&&owner!=="shared")body.set("customerId",owner);return api<Model>("/api/models/upload",{method:"POST",body})},onSuccess:model=>{queryClient.invalidateQueries({queryKey:["models"]});setSelected(model)}});
  const preview=useMutation({mutationFn:({id,file}:{id:string;file:File})=>{const body=new FormData();body.set("file",file);return api<{previewUrl:string}>(`/api/models/${id}/preview`,{method:"POST",body})},onSuccess:()=>queryClient.invalidateQueries({queryKey:["models"]})});
  return <>
    <PageHeader eyebrow="БИБЛИОТЕКА" title="3D-модели клиентов" description="STL, OBJ и 3MF привязаны к клиенту, доступны в заказе, Telegram и для скачивания." actions={<label className="customer-filter"><User size={15}/><select value={customerId} onChange={event=>{setCustomerId(event.target.value);setSelected(null)}}><option value="all">Все библиотеки</option><option value="shared">Общие модели</option>{customers.data?.map(customer=><option key={customer.id} value={customer.id}>{customer.name}</option>)}</select></label>}/>
    <div className="upload-zone"><input id="model-file" type="file" accept=".stl,.obj,.3mf" onChange={event=>{const file=event.target.files?.[0];if(file)upload.mutate({file,owner:customerId})}}/><label htmlFor="model-file"><span className="upload-icon"><FileUp/></span><h3>{upload.isPending?"Загружаем и анализируем…":"Добавить модель в выбранную библиотеку"}</h3><p>STL, OBJ, 3MF · до 200 МБ · {customerId==="all"||customerId==="shared"?"общая библиотека":customers.data?.find(customer=>customer.id===customerId)?.name}</p></label>{upload.error&&<ErrorNote error={upload.error}/>}</div>
    <div className="model-layout"><div className="model-list">{visible.map(model=><button className={`model-row ${active?.id===model.id?"active":""}`} key={model.id} onClick={()=>setSelected(model)}><strong><Layers3 size={14}/>{model.name}</strong><small>{model.format} · {(model.fileSizeBytes/1024/1024).toFixed(2)} МБ · {model.customerName||"Общая"}</small></button>)}</div>
      <div>{active?<><ModelViewer id={active.id} format={active.format}/><div className="model-actions"><button className="button" onClick={()=>downloadAuthenticated(active.fileUrl,active.originalFilename)}><Download size={15}/>Скачать модель</button><label className="button"><Camera size={15}/>{active.previewUrl?"Заменить фото":"Добавить фото для Telegram"}<input type="file" accept=".png,.jpg,.jpeg,.webp" hidden onChange={event=>{const file=event.target.files?.[0];if(file)preview.mutate({id:active.id,file})}}/></label><span className={active.previewUrl?"preview-ready":"preview-missing"}>{active.previewUrl?"Фото готово для бота":"Фото не добавлено"}</span></div>{preview.error&&<ErrorNote error={preview.error}/>}<div className="specs settings-section"><div><span>Размеры</span><strong>{active.dimensionsXmm?.toFixed(1)??"—"} × {active.dimensionsYmm?.toFixed(1)??"—"} × {active.dimensionsZmm?.toFixed(1)??"—"} мм</strong></div><div><span>Объём</span><strong>{active.volumeCm3?.toFixed(2)??"—"} см³</strong></div><div><span>Треугольники</span><strong>{active.triangleCount?.toLocaleString("ru")??"—"}</strong></div><div><span>Владелец</span><strong>{active.customerName||"Общая библиотека"}</strong></div><div><span>Файл</span><strong>{active.originalFilename}</strong></div></div></>:<div className="empty data-card">В этой библиотеке пока нет моделей</div>}</div>
    </div>
  </>;
}
