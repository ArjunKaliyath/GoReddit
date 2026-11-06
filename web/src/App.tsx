import { api } from "./api";
import { useForum, type View } from "./hooks/useForum";
import { Auth } from "./components/Auth";
import { PostCard } from "./components/PostCard";
import { Communities } from "./components/Communities";
import { Compose } from "./components/Compose";
import { Messages } from "./components/Messages";
const nav: { view: View; icon: string; label: string }[] = [
  { view: "feed", icon: "⌂", label: "Home feed" },
  { view: "explore", icon: "↗", label: "Explore posts" },
  { view: "communities", icon: "◉", label: "Communities" },
  { view: "messages", icon: "✉", label: "Messages" },
  { view: "people", icon: "♧", label: "People" },
];
export default function App() {
  const {
    user,
    setUser,
    initializing,
    view,
    setView,
    error,
    setError,
    busy,
    loading,
    communities,
    joined,
    posts,
    messages,
    people,
    following,
    refresh,
    act,
  } = useForum();
  return (
    <>
      <header className="topbar">
        <a href="/" className="brand">
          <span className="brand-mark">
            g<span>•</span>
          </span>
          GoReddit<span className="brand-dot">.</span>
        </a>
        <span className="tagline">A little curiosity goes a long way.</span>
        {user && (
          <div className="account">
            <span className="avatar">
              {user.username.charAt(0).toUpperCase()}
            </span>
            <span>u/{user.username}</span>
            <button
              className="quiet"
              disabled={busy}
              onClick={() =>
                act(async () => {
                  await api("/logout", {});
                  setUser(null);
                  setView("feed");
                })
              }
            >
              Sign out
            </button>
          </div>
        )}
      </header>
      {error && (
        <div className="error" role="alert">
          {error}
          <button aria-label="Dismiss error" onClick={() => setError("")}>
            ×
          </button>
        </div>
      )}
      {initializing ? (
        <main className="empty" role="status">
          Getting things ready…
        </main>
      ) : !user ? (
        <main>
          <Auth onLogin={setUser} act={act} busy={busy} />
        </main>
      ) : (
        <div className="app-shell">
          <aside className="sidebar">
            <div className="eyebrow">YOUR DAILY DOSE</div>
            <nav>
              {nav.map((n) => (
                <button
                  key={n.view}
                  aria-current={view === n.view ? "page" : undefined}
                  className={view === n.view ? "active" : ""}
                  onClick={() => setView(n.view)}
                >
                  <span aria-hidden="true">{n.icon}</span>
                  {n.label}
                </button>
              ))}
            </nav>
            <div className="sidebar-divider" />
            <div className="eyebrow">YOUR COMMUNITIES</div>
            {joined.map((c) => (
              <button
                className="joined-link"
                key={c.id}
                onClick={() => setView("communities")}
              >
                <span className="community-icon">
                  {c.name.charAt(0).toUpperCase()}
                </span>
                r/{c.name}
              </button>
            ))}
            {joined.length === 0 && (
              <p className="muted small">
                Find a community that feels like you.
              </p>
            )}
            <button className="link" onClick={() => setView("communities")}>
              + Discover communities
            </button>
            <div className="sidebar-footer">
              Built for people.
              <br />
              Powered by conversation.
            </div>
          </aside>
          <main className="workspace" aria-busy={loading}>
            <div className="content">
              {(view === "feed" || view === "explore") && (
                <>
                  <div className="section-heading">
                    <div className="eyebrow">
                      {view === "feed"
                        ? "YOUR PEOPLE. YOUR PERSPECTIVE."
                        : "FOLLOW YOUR CURIOSITY"}
                    </div>
                    <h1>
                      {view === "feed"
                        ? `Welcome back, ${user.username}.`
                        : "Worth a conversation."}
                    </h1>
                    <p className="muted">
                      {view === "feed"
                        ? "The latest from the communities you call home."
                        : "Discover what’s bringing people together."}
                    </p>
                  </div>
                  <div className="feed-toolbar">
                    <strong>
                      {view === "feed" ? "Your home feed" : "Top posts"}
                    </strong>
                    <button
                      className="quiet"
                      disabled={busy || loading}
                      onClick={() => act(refresh)}
                    >
                      ↻ Refresh
                    </button>
                    <button
                      className="primary"
                      onClick={() => setView("compose")}
                    >
                      + Create post
                    </button>
                  </div>
                  {loading && (
                    <p role="status" className="muted">
                      Loading conversations…
                    </p>
                  )}
                  {!loading && posts.length === 0 && (
                    <section className="card empty">
                      <span className="empty-icon">☀</span>
                      <h2>A fresh start for your feed</h2>
                      <p className="muted">
                        Join a community, share an idea, and make yourself at
                        home.
                      </p>
                      <button
                        className="primary"
                        onClick={() => setView("communities")}
                      >
                        Explore communities →
                      </button>
                    </section>
                  )}
                  {posts.map((p) => (
                    <PostCard
                      key={p.id}
                      post={p}
                      act={act}
                      refresh={refresh}
                      busy={busy}
                    />
                  ))}
                </>
              )}
              {view === "communities" && (
                <Communities
                  communities={communities}
                  joined={joined}
                  act={act}
                  refresh={refresh}
                  busy={busy}
                />
              )}{" "}
              {view === "compose" && (
                <Compose
                  communities={joined}
                  act={act}
                  busy={busy}
                  onDone={async () => {
                    setView("feed");
                    await refresh();
                  }}
                />
              )}
              {view === "messages" && (
                <Messages
                  messages={messages}
                  act={act}
                  refresh={refresh}
                  busy={busy}
                />
              )}{" "}
              {view === "people" && (
                <>
                  <div className="section-heading">
                    <div className="eyebrow">THE PEOPLE BEHIND THE POSTS</div>
                    <h1>Meet the community</h1>
                  </div>
                  {people.map((p) => (
                    <article className="card person" key={p.id}>
                      <span className="avatar">
                        {p.username.charAt(0).toUpperCase()}
                      </span>
                      <div>
                        <h2>u/{p.username}</h2>
                        <p className="muted">{p.karma} karma</p>
                      </div>
                      {p.id !== user.id && (
                        <button
                          className="secondary"
                          disabled={busy}
                          onClick={() =>
                            act(async () => {
                              await api(
                                `/users/${p.id}/${following.some((f) => f.id === p.id) ? "unsubscribe" : "subscribe"}`,
                                {},
                              );
                              await refresh();
                            })
                          }
                        >
                          {following.some((f) => f.id === p.id)
                            ? "Unfollow"
                            : "Follow"}
                        </button>
                      )}
                    </article>
                  ))}
                </>
              )}
            </div>
            <aside className="right-rail">
              <section className="welcome-panel">
                <span className="eyebrow">A PLACE TO BELONG</span>
                <h2>
                  Small moments.
                  <br />
                  Good conversations.
                </h2>
                <p>There’s a community for every curiosity. Find yours.</p>
                <button onClick={() => setView("communities")}>
                  Find your people →
                </button>
              </section>
              <section className="card rail-card">
                <h3>Make it a good space</h3>
                <p>✦ Be curious, be kind.</p>
                <p>✦ Share what moves you.</p>
                <p>✦ Make room for other voices.</p>
              </section>
              <p className="rail-footer">GoReddit · Made for connection</p>
            </aside>
          </main>
        </div>
      )}
    </>
  );
}
