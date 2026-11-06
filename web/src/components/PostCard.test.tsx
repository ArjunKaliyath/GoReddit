import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PostCard } from "./PostCard";
import type { Action, Post } from "../types";
const post: Post = {
  id: 7,
  title: "A good conversation",
  content: "Never render <script>alert(1)</script> as HTML",
  author_id: 1,
  author_name: "alice",
  subreddit_id: 2,
  subreddit_name: "golang",
  created_at: "2026-01-01T00:00:00Z",
  vote_count: { upvotes: 3, downvotes: 1 },
};
const act: Action = async (task) => {
  await task();
  return true;
};
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
describe("Post interactions", () => {
  it("renders untrusted content as text and sends the correct vote", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", fetch);
    const refresh = vi.fn().mockResolvedValue(undefined);
    render(<PostCard post={post} act={act} refresh={refresh} busy={false} />);
    expect(screen.getByText(post.content)).toBeInTheDocument();
    expect(document.querySelector("script")).toBeNull();
    await userEvent.click(
      screen.getByRole("button", { name: "Upvote A good conversation" }),
    );
    expect(fetch).toHaveBeenCalledWith(
      "/api/vote",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ target_id: 7, target_type: "post", value: 1 }),
      }),
    );
    expect(refresh).toHaveBeenCalled();
  });
  it("loads a conversation and persists a reply to the selected parent", async () => {
    const comments = [
      {
        id: 9,
        content: "First thought",
        author_username: "bobby",
        parent_comment_id: null,
      },
    ];
    const fetch = vi.fn().mockImplementation(async () => ({
      ok: true,
      json: async () => comments,
    }));
    vi.stubGlobal("fetch", fetch);
    render(
      <PostCard post={post} act={act} refresh={async () => {}} busy={false} />,
    );
    await userEvent.click(screen.getByRole("button", { name: /Conversation/ }));
    expect(await screen.findByText("First thought")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Reply" }));
    await userEvent.type(
      screen.getByLabelText("Your comment"),
      "A nested reply",
    );
    await userEvent.click(screen.getByRole("button", { name: "Post comment" }));
    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        "/api/comments",
        expect.objectContaining({
          body: JSON.stringify({
            post_id: 7,
            content: "A nested reply",
            parent_comment_id: 9,
          }),
        }),
      ),
    );
  });
});
