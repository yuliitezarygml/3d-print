"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, ExternalLink, Plus, Printer as PrinterIcon, Search, Zap } from "lucide-react";
import { api, money } from "@/lib/api";
import { ErrorNote, Modal, PageHeader, Status } from "@/components/ui";

type Printer = { id:string; name:string; manufacturer:string; model:string; serialNumber?:string; status:string; buildXmm:number; buildYmm:number; buildZmm:number; nozzleMm:number; powerWatts:number; purchasePrice:number; depreciationHours:number; totalHours:number; location?:string; catalogKey?:string; imageUrl?:string };
type CatalogModel = { key:string; manufacturer:string; model:string; fullName:string; technology:string; nozzleDiameters:number[]; buildXmm:number; buildYmm:number; buildZmm:number; imageUrl?:string; profileUrl:string; defaultMaterials:string[]; sources:{name:string}[] };
type Catalog = { total:number; generatedAt:string; sources:{name:string;repository:string;revision:string;license:string}[]; models:CatalogModel[] };

export default function Printers() {
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();
  const printers = useQuery({ queryKey:["printers"], queryFn:()=>api<Printer[]>("/api/printers") });
  const catalog = useQuery({ queryKey:["printer-catalog"], queryFn:()=>api<Catalog>("/api/printer-catalog") });
  const add = useMutation({ mutationFn:(body:Record<string,unknown>)=>api("/api/printers",{method:"POST",body:JSON.stringify(body)}), onSuccess:()=>{ queryClient.invalidateQueries({queryKey:["printers"]}); setOpen(false); } });
  return <>
    <PageHeader eyebrow="ПАРК ОБОРУДОВАНИЯ" title="3D-принтеры" description={`Фото, загрузка, мощность и стоимость часа. В справочнике ${catalog.data?.total??"…"} моделей.`} actions={<button className="button primary" onClick={()=>setOpen(true)}><Plus size={17}/>Добавить из справочника</button>}/>
    <div className="cards-grid">{printers.data?.map(printer=><article className="item-card printer-card" key={printer.id}>
      <div className="printer-cover" style={printer.imageUrl?{backgroundImage:`url("${printer.imageUrl}")`}:undefined}>{!printer.imageUrl&&<PrinterIcon size={42}/>}<Status value={printer.status}/></div>
      <h3>{printer.manufacturer} {printer.name}</h3><p>{printer.model} · {printer.location||"Мастерская"}</p>
      <div className="specs"><div><span>Рабочая область</span><strong>{printer.buildXmm} × {printer.buildYmm} × {printer.buildZmm} мм</strong></div><div><span>Сопло</span><strong>{printer.nozzleMm} мм</strong></div><div><span><Zap size={10}/> Мощность</span><strong>{printer.powerWatts} Вт</strong></div><div><span>Наработка</span><strong>{printer.totalHours.toFixed(1)} ч</strong></div><div><span>Стоимость</span><strong>{money(printer.purchasePrice)}</strong></div><div><span>Амортизация</span><strong>{(printer.purchasePrice/printer.depreciationHours).toFixed(2)} MDL/ч</strong></div></div>
    </article>)}</div>
    {open&&<Modal title="Добавить 3D-принтер" size="wide" onClose={()=>setOpen(false)}><CatalogPrinterForm catalog={catalog.data} busy={add.isPending} error={add.error} onSubmit={value=>add.mutate(value)} onCancel={()=>setOpen(false)}/></Modal>}
  </>;
}

function CatalogPrinterForm({catalog,onSubmit,onCancel,busy,error}:{catalog?:Catalog;onSubmit:(value:Record<string,unknown>)=>void;onCancel:()=>void;busy:boolean;error:unknown}) {
  const [search,setSearch]=useState("");
  const [selectedKey,setSelectedKey]=useState("");
  const matches=useMemo(()=>{const needle=search.trim().toLowerCase();return (catalog?.models??[]).filter(model=>!needle||`${model.manufacturer} ${model.model}`.toLowerCase().includes(needle)).slice(0,80)},[catalog,search]);
  const selected=catalog?.models.find(model=>model.key===selectedKey);
  function submit(event:React.FormEvent<HTMLFormElement>){event.preventDefault();if(!selected)return;const data=new FormData(event.currentTarget);onSubmit({catalogKey:selected.key,name:data.get("name"),manufacturer:selected.manufacturer,model:selected.model,status:"IDLE",buildXmm:selected.buildXmm,buildYmm:selected.buildYmm,buildZmm:selected.buildZmm,nozzleMm:Number(data.get("nozzle")),powerWatts:Number(data.get("power")),purchasePrice:Number(data.get("price")),depreciationHours:Number(data.get("depreciation")),location:data.get("location")||null})}
  return <form onSubmit={submit}>
    <div className="catalog-layout"><div className="catalog-browser"><label className="catalog-search"><Search size={16}/><input value={search} onChange={event=>setSearch(event.target.value)} placeholder="Bambu Lab, Prusa, Creality…"/></label><div className="catalog-list">{matches.map(model=><button type="button" className={`catalog-option ${selectedKey===model.key?"active":""}`} key={model.key} onClick={()=>setSelectedKey(model.key)}><span className="catalog-thumb" style={model.imageUrl?{backgroundImage:`url("${model.imageUrl}")`}:undefined}/><span><strong>{model.model}</strong><small>{model.manufacturer} · {model.buildXmm||"—"}×{model.buildYmm||"—"}×{model.buildZmm||"—"} мм</small></span>{selectedKey===model.key&&<Check size={16}/>}</button>)}</div></div>
      <div className="catalog-detail">{selected?<><div className="catalog-hero" style={selected.imageUrl?{backgroundImage:`url("${selected.imageUrl}")`}:undefined}/><p className="eyebrow">{selected.manufacturer}</p><h3>{selected.model}</h3><p>{selected.technology} · сопла {selected.nozzleDiameters.join(" / ")} мм</p><div className="catalog-specs"><span>Область печати<strong>{selected.buildXmm||"—"} × {selected.buildYmm||"—"} × {selected.buildZmm||"—"} мм</strong></span><span>Материалы<strong>{selected.defaultMaterials.slice(0,4).join(", ")||"По профилю производителя"}</strong></span></div><a href={selected.profileUrl} target="_blank" rel="noreferrer" className="source-link">Открыть исходный профиль <ExternalLink size={13}/></a><div className="form-grid compact"><label className="field full">Название в мастерской<input name="name" defaultValue={selected.model} required/></label><label className="field">Сопло<select name="nozzle" defaultValue={selected.nozzleDiameters.includes(.4)?.4:selected.nozzleDiameters[0]}>{selected.nozzleDiameters.map(value=><option key={value} value={value}>{value} мм</option>)}</select></label><label className="field">Средняя мощность, Вт<input name="power" type="number" min="0" defaultValue="250" required/></label><label className="field">Цена покупки, MDL<input name="price" type="number" min="0" step=".01" defaultValue="0"/></label><label className="field">Местоположение<input name="location" placeholder="Стеллаж A"/></label><label className="field full">Ресурс амортизации, ч<input name="depreciation" type="number" min="1" defaultValue="5000" required/></label></div></>:<div className="catalog-placeholder"><PrinterIcon size={42}/><strong>Выберите модель</strong><span>Фото и размеры заполнятся автоматически</span></div>}</div></div>
    {Boolean(error)&&<ErrorNote error={error}/>}<div className="form-actions"><button type="button" className="button" onClick={onCancel}>Отмена</button><button className="button primary" disabled={busy||!selected}>{busy?"Сохраняем…":"Добавить принтер"}</button></div>
    <p className="catalog-attribution">Профили и изображения: {catalog?.sources.map(source=>source.name).join(" + ")}. Лицензия AGPL‑3.0; ссылка на исходный профиль сохранена для каждой модели.</p>
  </form>;
}
