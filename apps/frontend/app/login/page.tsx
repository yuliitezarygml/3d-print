"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { Box, ArrowRight, Zap } from "lucide-react";
import { API_URL, session } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter(); const [email,setEmail]=useState("admin@printforge.local"); const [password,setPassword]=useState("admin12345"); const [error,setError]=useState(""); const [busy,setBusy]=useState(false);
  async function submit(e:React.FormEvent){e.preventDefault();setBusy(true);setError("");try{const res=await fetch(`${API_URL}/api/auth/login`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({email,password})});const body=await res.json();if(!res.ok)throw new Error(body.error);session.set(body);router.replace("/dashboard")}catch(e){setError(e instanceof Error?e.message:"Не удалось войти")}finally{setBusy(false)}}
  return <main className="login-page"><section className="login-art"><div className="brand"><span className="brand-mark"><Box size={21}/></span> PrintForge</div><div className="art-copy"><p className="eyebrow"><Zap size={15}/> WORKSHOP OS</p><h1>Ваша мастерская.<br/><span>Под полным контролем.</span></h1><p>От грамма пластика до стоимости каждого ватта — всё видно до запуска печати.</p></div><div className="art-meter"><div><span>Энергия сегодня</span><strong>12.48 кВт·ч</strong></div><div className="spark"><i/><i/><i/><i/><i/><i/><i/></div></div></section><section className="login-panel"><form className="login-card" onSubmit={submit}><div><p className="eyebrow">ДОБРО ПОЖАЛОВАТЬ</p><h2>Вход в мастерскую</h2><p className="muted">Введите данные администратора</p></div><label>Email<input type="email" value={email} onChange={e=>setEmail(e.target.value)} autoComplete="email" required/></label><label>Пароль<input type="password" value={password} onChange={e=>setPassword(e.target.value)} autoComplete="current-password" required/></label>{error&&<p className="form-error">{error}</p>}<button className="button primary wide" disabled={busy}>{busy?"Входим…":"Войти"}<ArrowRight size={17}/></button><p className="demo-hint">Демо: admin@printforge.local / admin12345</p></form></section></main>
}

