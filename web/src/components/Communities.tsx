import type { FormEvent } from "react";
import { api } from "../api";
import type { Action, Community } from "../types";
export function Communities({
  communities,
  joined,
  act,
  refresh,
  busy,
}: {
  communities: Community[];
  joined: Community[];
  act: Action;
  refresh: () => Promise<void>;
  busy: boolean;
}) {
  async function create(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form);
    if (
      await act(async () => {
        await api("/subreddits", {
          name: data.get("name"),
          description: data.get("description"),
        });
        await refresh();
      })
    )
      form.reset();
  }
  return (
    <>
      <div className="section-heading">
        <div className="eyebrow">FIND YOUR CORNER</div>
        <h1>Communities</h1>
        <p className="muted">
          Big interests. Small discoveries. A place for all of them.
        </p>
      </div>
      <div className="community-grid">
        {communities.map((c) => (
          <article className="card community-card" key={c.id}>
            <span className="community-icon large">
              {c.name.charAt(0).toUpperCase()}
            </span>
            <h2>r/{c.name}</h2>
            <p>{c.description}</p>
            <button
              disabled={busy}
              className={
                joined.some((j) => j.id === c.id) ? "secondary" : "primary"
              }
              onClick={() =>
                act(async () => {
                  await api(
                    `/subreddits/${c.id}/${joined.some((j) => j.id === c.id) ? "leave" : "join"}`,
                    {},
                  );
                  await refresh();
                })
              }
            >
              {joined.some((j) => j.id === c.id)
                ? "Leave community"
                : "Join community"}
            </button>
          </article>
        ))}
      </div>
      {communities.length === 0 && (
        <div className="empty card">
          No communities yet. Create the first one below.
        </div>
      )}
      <section className="card form-card">
        <h2>Start something good</h2>
        <p className="muted">
          Create a community around something you care about.
        </p>
        <form onSubmit={create}>
          <label>
            Community name
            <input
              name="name"
              required
              maxLength={64}
              placeholder="e.g. weekendreads"
            />
          </label>
          <label>
            Description
            <textarea
              name="description"
              required
              maxLength={2000}
              placeholder="What brings your community together?"
            />
          </label>
          <button disabled={busy} className="primary">
            Create community
          </button>
        </form>
      </section>
    </>
  );
}
