const COLORS = {
  red: "#e53935", blue: "#1e88e5", green: "#43a047", yellow: "#fdd835",
  purple: "#8e24aa", orange: "#fb8c00", pink: "#ec407a", cyan: "#00acc1",
  gray: "#757575", brown: "#6d4c41", lime: "#c0ca33", teal: "#00897b",
};

let gameId = null;
let selected = null;

const statusEl = document.getElementById("status");
const boardEl = document.getElementById("board");
const levelSelect = document.getElementById("levelSelect");

async function gql(query, variables) {
  const res = await fetch("/query", {
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

function setStatus(text, cls) {
  statusEl.textContent = text;
  statusEl.className = cls || "";
}

function renderBoard(game) {
  boardEl.innerHTML = "";
  game.tubes.forEach((tube, i) => {
    const wrap = document.createElement("div");
    wrap.className = "tube-wrap";

    const tubeEl = document.createElement("div");
    tubeEl.className = "tube" + (selected === i ? " selected" : "");
    tubeEl.dataset.index = i;
    tubeEl.addEventListener("click", () => onTubeClick(i));

    tube.forEach((color) => {
      const seg = document.createElement("div");
      seg.className = "segment";
      seg.style.background = COLORS[color] || color;
      tubeEl.appendChild(seg);
    });

    const label = document.createElement("div");
    label.className = "tube-label";
    label.textContent = i + 1;

    wrap.appendChild(tubeEl);
    wrap.appendChild(label);
    boardEl.appendChild(wrap);
  });

  if (game.solved) {
    setStatus("YOU WIN", "win");
  } else if (game.stuck) {
    setStatus("STUCK - no legal moves left", "stuck");
  } else {
    setStatus(`moves: ${game.moves}`);
  }
}

async function onTubeClick(i) {
  if (!gameId) return;
  if (selected === null) {
    selected = i;
    highlightSelected();
    return;
  }
  if (selected === i) {
    selected = null;
    highlightSelected();
    return;
  }
  const from = selected;
  selected = null;
  try {
    const data = await gql(
      `mutation($g: ID!, $f: Int!, $t: Int!) { move(gameId: $g, from: $f, to: $t) { levelId capacity tubes moves solved stuck } }`,
      { g: gameId, f: from + 1, t: i + 1 }
    );
    renderBoard(data.move);
  } catch (e) {
    setStatus(e.message, "err");
    highlightSelected();
  }
}

function highlightSelected() {
  document.querySelectorAll(".tube").forEach((el) => {
    el.classList.toggle("selected", Number(el.dataset.index) === selected);
  });
}

async function loadLevels() {
  const data = await gql(`{ levels { id difficulty } }`, {});
  levelSelect.innerHTML = "";
  data.levels.forEach((lvl) => {
    const opt = document.createElement("option");
    opt.value = lvl.id;
    opt.textContent = `Level ${lvl.id} (${lvl.difficulty.toLowerCase()})`;
    levelSelect.appendChild(opt);
  });
}

async function newGame() {
  selected = null;
  const levelId = Number(levelSelect.value);
  try {
    const data = await gql(
      `mutation($l: Int!) { newGame(levelId: $l) { id levelId capacity tubes moves solved stuck } }`,
      { l: levelId }
    );
    gameId = data.newGame.id;
    renderBoard(data.newGame);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function resetGame() {
  if (!gameId) return newGame();
  selected = null;
  try {
    const data = await gql(
      `mutation($g: ID!) { resetGame(gameId: $g) { levelId capacity tubes moves solved stuck } }`,
      { g: gameId }
    );
    renderBoard(data.resetGame);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function undo() {
  if (!gameId) return;
  selected = null;
  try {
    const data = await gql(
      `mutation($g: ID!) { undo(gameId: $g) { levelId capacity tubes moves solved stuck } }`,
      { g: gameId }
    );
    renderBoard(data.undo);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

document.getElementById("newGameBtn").addEventListener("click", newGame);
document.getElementById("resetBtn").addEventListener("click", resetGame);
document.getElementById("undoBtn").addEventListener("click", undo);

loadLevels().then(newGame);
