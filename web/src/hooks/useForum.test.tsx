import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import * as apiModule from "../api";
import { useForum } from "./useForum";
afterEach(() => vi.restoreAllMocks());
it("clears private data on logout and ignores an old in-flight refresh", async () => {
  vi.spyOn(apiModule, "api").mockRejectedValue(
    new apiModule.APIError(401, "Sign in"),
  );
  let finish: (value: never[]) => void = () => {};
  const pending = new Promise<never[]>((resolve) => {
    finish = resolve;
  });
  vi.spyOn(apiModule, "list").mockImplementation((path) =>
    path === "/feed" ? pending : Promise.resolve([]),
  );
  const { result } = renderHook(() => useForum());
  await waitFor(() => expect(result.current.initializing).toBe(false));
  act(() => result.current.setUser({ id: 1, username: "alice", karma: 0 }));
  await waitFor(() => expect(result.current.loading).toBe(true));
  act(() => result.current.setUser(null));
  await act(async () =>
    finish([{ id: 1, title: "Private data from old session" }] as never[]),
  );
  expect(result.current.posts).toEqual([]);
  expect(result.current.messages).toEqual([]);
  expect(result.current.user).toBeNull();
});
