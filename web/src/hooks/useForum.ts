import { useCallback, useEffect, useRef, useState } from "react";
import { api, APIError, list } from "../api";
import type { User, Community, Post, Message, Action } from "../types";
export type View =
  "feed" | "explore" | "communities" | "compose" | "messages" | "people";
export function useForum() {
  const [user, setUser] = useState<User | null>(null),
    [initializing, setInitializing] = useState(true),
    [view, setView] = useState<View>("feed");
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(false);
  const [communities, setCommunities] = useState<Community[]>([]),
    [joined, setJoined] = useState<Community[]>([]),
    [posts, setPosts] = useState<Post[]>([]),
    [messages, setMessages] = useState<Message[]>([]),
    [people, setPeople] = useState<User[]>([]),
    [following, setFollowing] = useState<User[]>([]);
  const refreshVersion = useRef(0);
  const refresh = useCallback(async () => {
    const version = ++refreshVersion.current;
    setLoading(true);
    try {
      const [c, j, p, m, u, f] = await Promise.all([
        list<Community>("/subreddits"),
        list<Community>("/subreddits/joined"),
        list<Post>(view === "feed" ? "/feed" : "/posts/top"),
        list<Message>("/messages"),
        list<User>("/users/top"),
        list<User>("/subscriptions"),
      ]);
      if (version !== refreshVersion.current) return;
      setCommunities(c);
      setJoined(j);
      setPosts(p);
      setMessages(m);
      setPeople(u);
      setFollowing(f);
    } catch (error) {
      if (version === refreshVersion.current) throw error;
    } finally {
      if (version === refreshVersion.current) setLoading(false);
    }
  }, [view]);
  const act: Action = async (task) => {
    setBusy(true);
    setError("");
    try {
      await task();
      return true;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong");
      if (e instanceof APIError && e.status === 401) setUser(null);
      return false;
    } finally {
      setBusy(false);
    }
  };
  useEffect(() => {
    let active = true;
    api<User>("/me")
      .then((u) => {
        if (active) setUser(u);
      })
      .catch((e) => {
        if (active && (!(e instanceof APIError) || e.status !== 401))
          setError("Unable to connect to the server. Refresh to retry.");
      })
      .finally(() => {
        if (active) setInitializing(false);
      });
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    if (!user) {
      setCommunities([]);
      setJoined([]);
      setPosts([]);
      setMessages([]);
      setPeople([]);
      setFollowing([]);
      setLoading(false);
    } else
      refresh().catch((e) => {
        setError(e.message);
        if (e instanceof APIError && e.status === 401) setUser(null);
      });
    return () => {
      refreshVersion.current++;
    };
  }, [user, refresh]);

  return {
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
  };
}
