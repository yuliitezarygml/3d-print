"use client";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState, useSyncExternalStore } from "react";
import { useQuery } from "@tanstack/react-query";
import { Box, LayoutDashboard, Printer, PackageOpen, ShoppingBag, Layers3, ListTodo, Users, Settings, Search, LogOut, Moon, Sun, Menu, X } from "lucide-react";
import { api, session } from "@/lib/api";

const nav = [
  { href:"/dashboard",label:"Обзор",icon:LayoutDashboard },{ href:"/orders",label:"Заказы",icon:ShoppingBag },{ href:"/jobs",label:"Очередь печати",icon:ListTodo },{ href:"/models",label:"3D-модели",icon:Layers3 },{ href:"/inventory",label:"Склад",icon:PackageOpen },{ href:"/printers",label:"Принтеры",icon:Printer },{ href:"/customers",label:"Клиенты",icon:Users },{ href:"/settings",label:"Настройки",icon:Settings },
];

const subscribeToMount = () => () => undefined;

export function AppShell({children}:{children:React.ReactNode}){
  const path=usePathname();const router=useRouter();const mounted=useSyncExternalStore(subscribeToMount,()=>true,()=>false);const [open,setOpen]=useState(false);
  const settings=useQuery({queryKey:["settings"],queryFn:()=>api<{electricityPricePerKwh:number}>("/api/settings"),enabled:mounted&&Boolean(session.get())});
  useEffect(()=>{if(!mounted)return;if(!session.get()){router.replace("/login");return}const saved=localStorage.getItem("printforge.theme");const value=saved==="dark"||(saved===null&&matchMedia("(prefers-color-scheme: dark)").matches);document.documentElement.dataset.theme=value?"dark":"light"},[mounted,router]);
  function toggleTheme(){const next=document.documentElement.dataset.theme!=="dark";document.documentElement.dataset.theme=next?"dark":"light";localStorage.setItem("printforge.theme",next?"dark":"light")}
  function logout(){session.clear();router.replace("/login")}
  if(!mounted||!session.get())return <div className="app-loading"><span className="brand-mark"><Box/></span></div>;
  return <div className="app-shell"><aside className={`sidebar ${open?"open":""}`}><div className="side-head"><Link className="brand" href="/dashboard"><span className="brand-mark"><Box size={20}/></span>PrintForge</Link><button className="icon-button close-side" onClick={()=>setOpen(false)} aria-label="Закрыть меню"><X/></button></div><nav>{nav.map(item=>{const Icon=item.icon;const active=path.startsWith(item.href);return <Link key={item.href} href={item.href} className={active?"active":""} onClick={()=>setOpen(false)}><Icon size={18}/><span>{item.label}</span></Link>})}</nav><div className="side-bottom"><div className="energy-chip"><span><i/> Энергия</span><strong>{settings.data?.electricityPricePerKwh??"—"} MDL / кВт·ч</strong><small>Текущий тариф</small></div><button onClick={logout}><LogOut size={18}/>Выйти</button></div></aside>{open&&<button className="scrim" aria-label="Закрыть меню" onClick={()=>setOpen(false)}/>}<div className="app-main"><header className="topbar"><button className="icon-button menu-button" aria-label="Открыть меню" onClick={()=>setOpen(true)}><Menu/></button><div className="command"><Search size={17}/><span>Поиск по мастерской…</span><kbd>⌘ K</kbd></div><div className="top-actions"><button className="icon-button theme-toggle" onClick={toggleTheme} aria-label="Сменить тему"><Moon className="theme-moon" size={18}/><Sun className="theme-sun" size={18}/></button><div className="user"><span>AD</span><div><strong>Administrator</strong><small>Администратор</small></div></div></div></header><div className="page-wrap">{children}</div></div></div>
}
