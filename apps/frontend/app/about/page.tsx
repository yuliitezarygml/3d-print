import Image from "next/image";
import Link from "next/link";
import type { Metadata } from "next";
import { ArrowDownRight, ArrowRight, Box, CircleCheck, Layers3, ScanLine, Sparkles } from "lucide-react";

import { CinematicVideo } from "@/components/cinematic-video";
import { PublicTrackingForm } from "@/components/public-tracking-form";

export const metadata: Metadata = {
  title: "О мастерской — PrintForge",
  description: "3D-печать прототипов, деталей и небольших серий с прозрачным расчётом стоимости и отслеживанием заказа.",
};

const steps = [
  { number: "01", title: "Загружаете модель", text: "STL, OBJ или 3MF — прямо с телефона или компьютера." },
  { number: "02", title: "Мы проверяем", text: "Оцениваем геометрию, материал, сроки и точную стоимость." },
  { number: "03", title: "Печатаем", text: "Вы видите статус заказа по персональному коду в реальном времени." },
  { number: "04", title: "Получаете", text: "Готовую деталь, аккуратно обработанную и упакованную." },
];

const principles = [
  { icon: ScanLine, title: "Точность до слоя", text: "Проверяем ориентацию, поддержки и качество поверхности до запуска печати." },
  { icon: CircleCheck, title: "Честная стоимость", text: "Материал, электричество и работа учтены в понятном расчёте без скрытых доплат." },
  { icon: Layers3, title: "История не теряется", text: "Модели, чеки и статусы остаются привязаны к заказу и доступны для скачивания." },
];

const printers = [
  { name: "Bambu Lab X1 Carbon", role: "Быстрая серийная печать", image: "/printer-catalog/bambu-lab-x1-carbon-7ec0af4643.png" },
  { name: "Prusa MK4", role: "Точные функциональные детали", image: "/printer-catalog/prusa-mk4-2daf026d57.png" },
  { name: "Creality K1", role: "Прототипы и крупные формы", image: "/printer-catalog/creality-k1-dbb27425cf.png" },
];

export default function AboutPage() {
  return (
    <main className="about-page">
      <section className="about-hero">
        <CinematicVideo className="about-hero-video" label="Работающая 3D-печатная мастерская" poster="/media/about-workshop-hero.jpg" src="/media/about-workshop-hero.webm" />
        <div className="about-hero-shade" />

        <header className="about-nav">
          <Link className="about-logo" href="/about"><span>3D</span> PRINT</Link>
          <nav aria-label="Навигация по странице">
            <a href="#story">О нас</a><a href="#process">Как работаем</a><a href="#equipment">Оборудование</a><a href="#tracking">Проверить заказ</a>
          </nav>
          <Link className="about-nav-cta" href="/request">Рассчитать печать <ArrowRight aria-hidden="true" /></Link>
        </header>

        <div className="about-hero-content">
          <p className="about-kicker"><span /> Мастерская цифрового производства</p>
          <h1>Идея становится<br /><em>вещью.</em></h1>
          <p className="about-hero-copy">Печатаем прототипы, детали и небольшие серии. Берём на себя путь от файла до готового изделия — прозрачно и бережно.</p>
          <div className="about-hero-actions">
            <Link className="about-button about-button-primary" href="/request">Загрузить модель <ArrowRight /></Link>
            <a className="about-text-link" href="#story">Познакомиться с нами <ArrowDownRight /></a>
          </div>
        </div>
        <div className="about-hero-note"><span>01</span><p>Производим локально<br />в Молдове</p></div>
      </section>

      <section className="about-story" id="story">
        <div className="about-section-mark">/ О мастерской</div>
        <div className="about-story-copy">
          <h2>Мы превращаем цифровые модели в <span>настоящие предметы.</span></h2>
          <p>Наша мастерская создана для людей, которым нужен не просто отпечаток, а предсказуемый результат. Мы проверяем каждый файл, подбираем материал и показываем весь путь заказа — от расчёта до выдачи.</p>
        </div>
        <div className="about-facts" aria-label="Факты о сервисе">
          <article><strong>387</strong><span>профилей материалов и принтеров</span></article>
          <article><strong>4</strong><span>формата 3D-моделей</span></article>
          <article><strong>100%</strong><span>истории заказа доступно клиенту</span></article>
        </div>
      </section>

      <section className="about-film" aria-label="Процесс 3D-печати">
        <CinematicVideo className="about-film-video" controls label="Крупный план процесса 3D-печати" poster="/media/about-print-process.jpg" src="/media/about-print-process.webm" />
        <div className="about-film-caption"><span>Внутри процесса</span><strong>Точность начинается<br />с первого слоя</strong></div>
        <div className="about-film-badge"><Sparkles /> Сделано послойно</div>
      </section>

      <section className="about-process" id="process">
        <div className="about-section-heading"><div className="about-section-mark">/ Как всё происходит</div><h2>От файла до готовой детали — <span>четыре понятных шага.</span></h2></div>
        <div className="about-steps">
          {steps.map((step) => <article key={step.number}><span>{step.number}</span><div><h3>{step.title}</h3><p>{step.text}</p></div></article>)}
        </div>
      </section>

      <section className="about-principles">
        <div className="about-principles-copy"><div className="about-section-mark">/ Наш подход</div><h2>Технология важна.<br /><span>Отношение — важнее.</span></h2><p>Мы построили сервис так, чтобы сложное производство ощущалось простым и понятным.</p></div>
        <div className="about-principle-grid">
          {principles.map(({ icon: Icon, title, text }) => <article key={title}><Icon aria-hidden="true" /><h3>{title}</h3><p>{text}</p></article>)}
        </div>
      </section>

      <section className="about-equipment" id="equipment">
        <div className="about-section-heading"><div className="about-section-mark">/ Оборудование</div><h2>Для каждой задачи — <span>свой характер.</span></h2></div>
        <div className="about-printer-grid">
          {printers.map((printer, index) => <article key={printer.name}><div className="about-printer-image"><Image alt={printer.name} fill sizes="(max-width: 900px) 100vw, 33vw" src={printer.image} /></div><div className="about-printer-meta"><span>0{index + 1}</span><div><h3>{printer.name}</h3><p>{printer.role}</p></div></div></article>)}
        </div>
      </section>

      <section className="about-tracking" id="tracking">
        <div><div className="about-section-mark">/ Где мой заказ?</div><h2>Следите за печатью<br />без звонков и ожидания.</h2><p>Введите персональный код из Telegram или чека — увидите модель, текущий этап, стоимость и доступные файлы.</p></div>
        <PublicTrackingForm />
      </section>

      <section className="about-final">
        <CinematicVideo className="about-final-video" label="3D-принтеры в мастерской" poster="/media/about-workshop-hero.jpg" src="/media/about-workshop-hero.webm" />
        <div className="about-final-shade" />
        <div className="about-final-copy"><Box aria-hidden="true" /><h2>Есть идея?<br /><span>Давайте напечатаем.</span></h2><Link className="about-button about-button-primary" href="/request">Получить расчёт <ArrowRight /></Link></div>
      </section>

      <footer className="about-footer"><Link className="about-logo" href="/about"><span>3D</span> PRINT</Link><p>Цифровое производство, которое можно понять.</p><Link href="/dashboard">Вход для мастерской</Link></footer>
    </main>
  );
}
