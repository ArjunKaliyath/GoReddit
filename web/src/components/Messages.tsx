import { useState, type FormEvent } from "react";
import { api } from "../api";
import type { Action, Message } from "../types";
export function Messages({
  messages,
  act,
  refresh,
  busy,
}: {
  messages: Message[];
  act: Action;
  refresh: () => Promise<void>;
  busy: boolean;
}) {
  const [sent, setSent] = useState(false);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setSent(false);
    const form = e.currentTarget;
    const d = new FormData(form);
    if (
      await act(async () => {
        await api("/messages", {
          to_username: d.get("recipient"),
          content: d.get("content"),
        });
        await refresh();
      })
    ) {
      form.reset();
      setSent(true);
    }
  }
  return (
    <>
      <div className="section-heading">
        <div className="eyebrow">KEEP THE CONNECTION GOING</div>
        <h1>Your inbox</h1>
      </div>
      <section className="card form-card">
        <h2>Send a message</h2>
        {sent && <p role="status">Message sent.</p>}
        <form onSubmit={submit}>
          <label>
            To
            <input
              name="recipient"
              placeholder="Recipient’s username"
              required
            />
          </label>
          <label>
            Message
            <textarea
              name="content"
              required
              maxLength={10000}
              placeholder="Say hello…"
            />
          </label>
          <button className="primary" disabled={busy}>
            Send message
          </button>
        </form>
      </section>
      {messages.length === 0 ? (
        <div className="empty card">
          Your inbox is quiet. New messages will appear here.
        </div>
      ) : (
        messages.map((m) => (
          <article className="card message" key={m.id}>
            <strong>u/{m.from_username}</strong>
            <time>{new Date(m.created_at).toLocaleString()}</time>
            <p>{m.content}</p>
          </article>
        ))
      )}
    </>
  );
}
