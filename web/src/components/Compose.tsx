import type { FormEvent } from "react";
import { api } from "../api";
import type { Action, Community } from "../types";
export function Compose({
  communities,
  act,
  onDone,
  busy,
}: {
  communities: Community[];
  act: Action;
  onDone: () => Promise<void>;
  busy: boolean;
}) {
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const d = new FormData(e.currentTarget);
    await act(async () => {
      await api("/posts", {
        title: d.get("title"),
        content: d.get("content"),
        subreddit_id: Number(d.get("community")),
      });
      await onDone();
    });
  }
  return (
    <section className="card form-card">
      <div className="eyebrow">SOMETHING ON YOUR MIND?</div>
      <h1>Create a post</h1>
      <p className="muted">
        A question, an idea, a story. Make it the start of a conversation.
      </p>
      <form onSubmit={submit}>
        <label>
          Community
          <select name="community" required defaultValue="">
            <option value="" disabled>
              Choose a community
            </option>
            {communities.map((c) => (
              <option value={c.id} key={c.id}>
                r/{c.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Title
          <input
            name="title"
            required
            maxLength={200}
            placeholder="Give your post a title"
          />
        </label>
        <label>
          Your post
          <textarea
            rows={8}
            name="content"
            required
            maxLength={10000}
            placeholder="Tell us more…"
          />
        </label>
        <button className="primary" disabled={busy || communities.length === 0}>
          Publish post →
        </button>
        {communities.length === 0 && <p>Join or create a community first.</p>}
      </form>
    </section>
  );
}
