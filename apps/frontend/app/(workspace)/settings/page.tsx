"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bot, ExternalLink, Save, ShieldCheck, Trash2, Zap } from "lucide-react";
import { api } from "@/lib/api";
import { ErrorNote, PageHeader } from "@/components/ui";

type Settings={companyName:string;currency:string;electricityPricePerKwh:number;machineRatePerHour:number;labourRatePerHour:number;defaultMarkupPercent:number;lowStockThresholdGrams:number;publicBaseUrl:string;telegramBotConfigured:boolean;telegramBotUsername?:string;telegramBotEnabled:boolean};
type CoreSettings=Pick<Settings,"companyName"|"currency"|"electricityPricePerKwh"|"machineRatePerHour"|"labourRatePerHour"|"defaultMarkupPercent"|"lowStockThresholdGrams">;

export default function SettingsPage(){const query=useQuery({queryKey:["settings"],queryFn:()=>api<Settings>("/api/settings")});if(!query.data)return <PageHeader title="Настройки" description="Загрузка…"/>;return <SettingsForm key={JSON.stringify(query.data)} initial={query.data}/>}

function SettingsForm({initial}:{initial:Settings}){
  const queryClient=useQueryClient();
  const [value,setValue]=useState<CoreSettings>(initial);
  const [telegram,setTelegram]=useState({token:"",enabled:initial.telegramBotEnabled,publicBaseUrl:initial.publicBaseUrl});
  const save=useMutation({mutationFn:(body:CoreSettings)=>api<Settings>("/api/settings",{method:"PUT",body:JSON.stringify(body)}),onSuccess:data=>{setValue(data);queryClient.setQueryData(["settings"],data)}});
  const saveTelegram=useMutation({mutationFn:(body:{token:string;enabled:boolean;publicBaseUrl:string;removeToken?:boolean})=>api<Settings>("/api/settings/telegram",{method:"PUT",body:JSON.stringify(body)}),onSuccess:data=>{setTelegram({token:"",enabled:data.telegramBotEnabled,publicBaseUrl:data.publicBaseUrl});queryClient.setQueryData(["settings"],data)}});
  function update<K extends keyof CoreSettings>(key:K,next:CoreSettings[K]){setValue(current=>({...current,[key]:next}))}
  return <>
    <PageHeader eyebrow="КОНФИГУРАЦИЯ" title="Настройки мастерской" description="Тарифы, публичная страница заказа и Telegram‑бот." actions={<button className="button primary" onClick={()=>save.mutate(value)} disabled={save.isPending}><Save size={16}/>{save.isPending?"Сохраняем…":"Сохранить тарифы"}</button>}/>
    <section className="settings-section"><h2>Основное</h2><p>Название мастерской и валюта финансовых расчётов.</p><div className="form-grid"><label className="field">Название<input value={value.companyName} onChange={event=>update("companyName",event.target.value)}/></label><label className="field">Валюта<select value={value.currency} onChange={event=>update("currency",event.target.value)}><option>MDL</option><option>EUR</option><option>USD</option><option>RON</option></select></label></div></section>
    <section className="settings-section"><h2><Zap size={17}/> Электричество и производство</h2><p>Значения фиксируются в задании: история не изменится после смены тарифа.</p><div className="form-grid"><NumberField label="Тариф за кВт·ч" hint="MDL / кВт·ч" step=".0001" value={value.electricityPricePerKwh} onChange={next=>update("electricityPricePerKwh",next)}/><NumberField label="Ставка станка" hint="MDL / час, без амортизации" value={value.machineRatePerHour} onChange={next=>update("machineRatePerHour",next)}/><NumberField label="Работа оператора" hint="MDL / час" value={value.labourRatePerHour} onChange={next=>update("labourRatePerHour",next)}/><NumberField label="Наценка по умолчанию" hint="%" step=".1" value={value.defaultMarkupPercent} onChange={next=>update("defaultMarkupPercent",next)}/></div></section>
    <section className="settings-section"><h2>Склад</h2><p>Порог, ниже которого катушка считается заканчивающейся.</p><div className="form-grid"><NumberField label="Низкий остаток" hint="грамм" step="1" value={value.lowStockThresholdGrams} onChange={next=>update("lowStockThresholdGrams",next)}/></div></section>
    <section className="settings-section telegram-settings"><div className="telegram-heading"><span className="telegram-icon"><Bot/></span><div><h2>Telegram‑бот для клиентов</h2><p>Клиент отправляет код заказа и получает статус, стоимость, фото и файл модели.</p></div><span className={initial.telegramBotConfigured?"telegram-connected":"telegram-disconnected"}><ShieldCheck size={14}/>{initial.telegramBotConfigured?`@${initial.telegramBotUsername||"бот подключён"}`:"Не подключён"}</span></div>
      <div className="telegram-help"><strong>Как подключить</strong><span>1. Создайте бота у <a href="https://t.me/BotFather" target="_blank" rel="noreferrer">@BotFather <ExternalLink size={12}/></a></span><span>2. Вставьте токен ниже — он будет проверен Telegram и сохранён зашифрованно.</span><span>3. Включите бота. Локально он работает через безопасный long polling, домен не нужен.</span></div>
      <div className="form-grid"><label className="field full">Токен бота<input type="password" autoComplete="off" value={telegram.token} onChange={event=>setTelegram(current=>({...current,token:event.target.value}))} placeholder={initial.telegramBotConfigured?"Оставьте пустым, чтобы сохранить текущий токен":"123456789:AA…"}/><small>Токен никогда не возвращается из API и не показывается после сохранения.</small></label><label className="field full">Публичный адрес сайта<input value={telegram.publicBaseUrl} onChange={event=>setTelegram(current=>({...current,publicBaseUrl:event.target.value}))} placeholder="http://localhost"/><small>После деплоя замените на домен Vercel.</small></label><label className="telegram-switch full"><input type="checkbox" checked={telegram.enabled} onChange={event=>setTelegram(current=>({...current,enabled:event.target.checked}))}/><span><strong>Бот включён</strong><small>Принимает код заказа и отправляет обновлённые данные.</small></span></label></div>
      {saveTelegram.error&&<ErrorNote error={saveTelegram.error}/>}<div className="telegram-actions"><button className="button primary" disabled={saveTelegram.isPending} onClick={()=>saveTelegram.mutate(telegram)}><Bot size={15}/>{saveTelegram.isPending?"Проверяем…":"Проверить и сохранить"}</button>{initial.telegramBotConfigured&&<button className="button danger" onClick={()=>saveTelegram.mutate({token:"",enabled:false,publicBaseUrl:telegram.publicBaseUrl,removeToken:true})}><Trash2 size={15}/>Отключить и удалить токен</button>}</div>
    </section>
    {save.error&&<ErrorNote error={save.error}/>}
  </>;
}

function NumberField({label,hint,value,step=".01",onChange}:{label:string;hint:string;value:number;step?:string;onChange:(value:number)=>void}){return <label className="field">{label}<input type="number" min="0" step={step} value={value} onChange={event=>onChange(Number(event.target.value))}/><small>{hint}</small></label>}
