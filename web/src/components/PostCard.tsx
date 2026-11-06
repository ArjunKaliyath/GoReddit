import { useState, type FormEvent } from "react";
import { api, list } from "../api";
import type { Action, Comment, Post } from "../types";
export function PostCard({
  post,
  act,
  refresh,
  busy,
}: {
  post: Post;
  act: Action;
  refresh: () => Promise<void>;
  busy: boolean;
}) {
  const [comments, setComments] = useState<Comment[] | null>(null);
  const [reply, setReply] = useState<number | null>(null);
  async function vote(value: number) {
    await act(async () => {
      await api("/vote", { target_id: post.id, target_type: "post", value });
      await refresh();
    });
  }
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const content = new FormData(form).get("content");
    if (
      await act(async () => {
        await api("/comments", {
          post_id: post.id,
          content,
          parent_comment_id: reply,
        });
        setComments(await list<Comment>(`/posts/${post.id}/comments`));
      })
    ) {
      form.reset();
      setReply(null);
    }
  }
  return (
    <article className="card post">
      <div className="post-meta">
        <span className="community-icon">
          {post.subreddit_name.charAt(0).toUpperCase()}
        </span>
        <strong>r/{post.subreddit_name}</strong>
        <span>· by u/{post.author_name}</span>
        <time dateTime={post.created_at}>
          {new Date(post.created_at).toLocaleDateString()}
        </time>
      </div>
      <h2>{post.title}</h2>
      <p className="post-body">{post.content}</p>
      <div className="post-actions">
        <div className="vote">
          <button
            aria-label={`Upvote ${post.title}`}
            disabled={busy}
            onClick={() => vote(1)}
          >
            ↑
          </button>
          <strong>{post.vote_count.upvotes - post.vote_count.downvotes}</strong>
          <button
            aria-label={`Downvote ${post.title}`}
            disabled={busy}
            onClick={() => vote(-1)}
          >
            ↓
          </button>
        </div>
        <button
          className="quiet"
          disabled={busy}
          onClick={() =>
            act(async () =>
              setComments(
                comments === null
                  ? await list<Comment>(`/posts/${post.id}/comments`)
                  : null,
              ),
            )
          }
        >
          ☷{" "}
          {comments === null ? "Conversation" : `${comments.length} comments`}
        </button>
      </div>
      {comments !== null && (
        <section className="comments" aria-label="Comments">
          {comments.length === 0 && (
            <p className="muted">Be the first to add a thought.</p>
          )}
          {comments.map((c) => (
            <div
              className={`comment ${c.parent_comment_id ? "reply" : ""}`}
              key={c.id}
            >
              <strong>u/{c.author_username}</strong>
              {c.parent_comment_id && (
                <small> ↳ reply to #{c.parent_comment_id}</small>
              )}
              <p>{c.content}</p>
              <button className="link" onClick={() => setReply(c.id)}>
                Reply
              </button>
            </div>
          ))}
          <form onSubmit={submit}>
            {reply !== null && (
              <p>
                Replying to #{reply}{" "}
                <button
                  type="button"
                  className="link"
                  onClick={() => setReply(null)}
                >
                  Cancel
                </button>
              </p>
            )}
            <label className="sr-only" htmlFor={`comment-${post.id}`}>
              Your comment
            </label>
            <textarea
              id={`comment-${post.id}`}
              name="content"
              placeholder="Add to the conversation…"
              required
              maxLength={10000}
            />
            <button disabled={busy} className="primary">
              Post comment
            </button>
          </form>
        </section>
      )}
    </article>
  );
}
