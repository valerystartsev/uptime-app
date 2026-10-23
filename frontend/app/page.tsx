import Link from "next/link";

export default function Home() {
  return (
    <main className="home-page">
      <section className="home-card">
        <p className="eyebrow">Uptime</p>
        <h1>Следите за доступностью сервисов</h1>
        <p>Создайте аккаунт или войдите в существующую защищённую сессию.</p>
        <div className="home-actions">
          <Link className="button" href="/register">
            Создать аккаунт
          </Link>
          <Link className="button button-secondary" href="/login">
            Войти
          </Link>
        </div>
      </section>
    </main>
  );
}
