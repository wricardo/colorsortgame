import { describe, it, expect, vi } from "vitest";
import { gql, statusFor, urlWithGameId, gameIdFromUrl, renderBoard } from "./logic.js";

describe("gql", () => {
  it("posts query/variables and returns data on success", async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      json: async () => ({ data: { foo: "bar" } }),
    });
    const data = await gql(fetchImpl, "{ foo }", { a: 1 });
    expect(data).toEqual({ foo: "bar" });
    expect(fetchImpl).toHaveBeenCalledWith(
      "/query",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: "{ foo }", variables: { a: 1 } }),
      })
    );
  });

  it("throws joined error messages when the response has errors", async () => {
    const fetchImpl = vi.fn().mockResolvedValue({
      json: async () => ({ errors: [{ message: "bad move" }, { message: "stuck" }] }),
    });
    await expect(gql(fetchImpl, "{ foo }", {})).rejects.toThrow("bad move; stuck");
  });
});

describe("statusFor", () => {
  it("reports a win", () => {
    expect(statusFor({ solved: true, stuck: false, moves: 3 })).toEqual({ text: "YOU WIN", cls: "win" });
  });

  it("reports stuck", () => {
    expect(statusFor({ solved: false, stuck: true, moves: 5 })).toEqual({
      text: "STUCK - no legal moves left",
      cls: "stuck",
    });
  });

  it("reports move count otherwise", () => {
    expect(statusFor({ solved: false, stuck: false, moves: 7 })).toEqual({ text: "moves: 7", cls: "" });
  });
});

describe("urlWithGameId / gameIdFromUrl", () => {
  it("adds the game param", () => {
    const url = urlWithGameId("https://example.com/", "abc-123");
    expect(url.searchParams.get("game")).toBe("abc-123");
  });

  it("removes the game param when id is falsy", () => {
    const url = urlWithGameId("https://example.com/?game=abc-123", null);
    expect(url.searchParams.has("game")).toBe(false);
  });

  it("reads the game param back out", () => {
    expect(gameIdFromUrl("https://example.com/?game=xyz")).toBe("xyz");
  });

  it("returns null when no game param present", () => {
    expect(gameIdFromUrl("https://example.com/")).toBeNull();
  });
});

describe("renderBoard", () => {
  it("renders one tube-wrap per tube with segments colored per tube contents", () => {
    document.body.innerHTML = `<div id="board"></div><div id="status"></div>`;
    const boardEl = document.getElementById("board");
    const statusEl = document.getElementById("status");
    const game = { tubes: [["red", "blue"], []], moves: 2, solved: false, stuck: false };

    renderBoard(document, boardEl, statusEl, game, null, () => {});

    const wraps = boardEl.querySelectorAll(".tube-wrap");
    expect(wraps.length).toBe(2);
    expect(wraps[0].querySelectorAll(".segment").length).toBe(2);
    expect(wraps[1].querySelectorAll(".segment").length).toBe(0);
    expect(statusEl.textContent).toBe("moves: 2");
  });

  it("marks the selected tube", () => {
    document.body.innerHTML = `<div id="board"></div><div id="status"></div>`;
    const boardEl = document.getElementById("board");
    const statusEl = document.getElementById("status");
    const game = { tubes: [[], []], moves: 0, solved: false, stuck: false };

    renderBoard(document, boardEl, statusEl, game, 1, () => {});

    const tubes = boardEl.querySelectorAll(".tube");
    expect(tubes[0].classList.contains("selected")).toBe(false);
    expect(tubes[1].classList.contains("selected")).toBe(true);
  });

  it("invokes onTubeClick with the tube index when clicked", () => {
    document.body.innerHTML = `<div id="board"></div><div id="status"></div>`;
    const boardEl = document.getElementById("board");
    const statusEl = document.getElementById("status");
    const game = { tubes: [[], []], moves: 0, solved: false, stuck: false };
    const onTubeClick = vi.fn();

    renderBoard(document, boardEl, statusEl, game, null, onTubeClick);
    boardEl.querySelectorAll(".tube")[1].dispatchEvent(new window.Event("click", { bubbles: true }));

    expect(onTubeClick).toHaveBeenCalledWith(1);
  });
});
