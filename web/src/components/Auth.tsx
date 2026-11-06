import { useState, type FormEvent } from "react";
import { api } from "../api";
import type { Action, User } from "../types";
export function Auth({
  onLogin,
  act,
  busy,
}: {
  onLogin: (u: User) => void;
  act: Action;
  busy: boolean;
}) {
  const [register, setRegister] = useState(false);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    await act(async () =>
      onLogin(
        await api<User>(register ? "/register" : "/login", {
          username: data.get("username"),
          password: data.get("password"),
        }),
      ),
    );
  }
  return (
    <div className="auth-layout">
      <section className="intro">
        <div className="eyebrow">GOOD CONVERSATIONS START HERE</div>
        <h1>
          Find your people.
          <br />
          <span>Share your world.</span>
        </h1>
        <p>
          A home for curious minds, shared interests, and the conversations that
          bring us together.
        </p>
        <div className="intro-tags">
          <span>◉ Communities</span>
          <span>↗ Fresh perspectives</span>
          <span>✦ Real connections</span>
        </div>
      </section>
      <section className="card auth-card">
        <div className="eyebrow">WELCOME TO GOREDDIT</div>
        <h2>{register ? "Make yourself at home" : "Good to see you again"}</h2>
        <p className="muted">
          {register
            ? "Create an account to join the conversation."
            : "Sign in to pick up the conversation."}
        </p>
        <form onSubmit={submit}>
          <label>
            Username
            <input
              name="username"
              autoComplete="username"
              required
              minLength={3}
              maxLength={32}
              pattern="[A-Za-z0-9_]+"
              placeholder="Your username"
            />
          </label>
          <label>
            Password
            <input
              name="password"
              type="password"
              autoComplete={register ? "new-password" : "current-password"}
              required
              minLength={register ? 8 : 1}
              maxLength={72}
              placeholder={register ? "At least 8 characters" : "Your password"}
            />
          </label>
          <button className="primary wide" disabled={busy}>
            {busy ? "Please wait…" : register ? "Create account" : "Sign in"}{" "}
            <span aria-hidden="true">→</span>
          </button>
        </form>
        <p className="switch">
          {register ? "Already a member?" : "New around here?"}{" "}
          <button className="link" onClick={() => setRegister(!register)}>
            {register ? "Sign in" : "Create an account"}
          </button>
        </p>
      </section>
    </div>
  );
}
