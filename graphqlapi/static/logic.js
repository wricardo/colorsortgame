export const COLORS = {
  red: "#e53935", blue: "#1e88e5", green: "#43a047", yellow: "#fdd835",
  purple: "#8e24aa", orange: "#fb8c00", pink: "#ec407a", cyan: "#00acc1",
  gray: "#757575", brown: "#6d4c41", lime: "#c0ca33", teal: "#00897b",
};

export async function gql(fetchImpl, query, variables) {
  const res = await fetchImpl("/query", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  const body = await res.json();
  if (body.errors && body.errors.length) {
    throw new Error(body.errors.map((e) => e.message).join("; "));
  }
  return body.data;
}

export function statusFor(game) {
  if (game.solved) return { text: "YOU WIN", cls: "win" };
  if (game.stuck) return { text: "STUCK - no legal moves left", cls: "stuck" };
  return { text: `moves: ${game.moves}`, cls: "" };
}

export function urlWithGameId(href, id) {
  const url = new URL(href);
  if (id) url.searchParams.set("game", id);
  else url.searchParams.delete("game");
  return url;
}

export function gameIdFromUrl(href) {
  return new URL(href).searchParams.get("game");
}

export function renderBoard(doc, boardEl, statusEl, game, selected, onTubeClick) {
  boardEl.innerHTML = "";
  game.tubes.forEach((tube, i) => {
    const wrap = doc.createElement("div");
    wrap.className = "tube-wrap";

    const tubeEl = doc.createElement("div");
    tubeEl.className = "tube" + (selected === i ? " selected" : "");
    tubeEl.dataset.index = i;
    tubeEl.addEventListener("click", () => onTubeClick(i));

    tube.forEach((color) => {
      const seg = doc.createElement("div");
      seg.className = "segment";
      seg.style.background = COLORS[color] || color;
      tubeEl.appendChild(seg);
    });

    const label = doc.createElement("div");
    label.className = "tube-label";
    label.textContent = i + 1;

    wrap.appendChild(tubeEl);
    wrap.appendChild(label);
    boardEl.appendChild(wrap);
  });

  const { text, cls } = statusFor(game);
  statusEl.textContent = text;
  statusEl.className = cls;
}
