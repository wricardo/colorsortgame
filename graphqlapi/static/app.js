import {
  gql,
  renderBoard,
  renderMoveLog,
  urlWithGameId,
  gameIdFromUrl,
  wsUrlFor,
  openGameSubscription,
} from "./logic.js";

let gameId = null;
let selected = null;
let currentGame = null;
let closeSubscription = null;
let levelsMap = {}; // levelId -> difficulty

const statusEl = document.getElementById("status");
const boardEl = document.getElementById("board");
const levelSelect = document.getElementById("levelSelect");
const gameIdInput = document.getElementById("gameIdInput");
const moveLogEl = document.getElementById("moveLog");
const levelDisplay = document.getElementById("levelDisplay");
const difficultyDisplay = document.getElementById("difficultyDisplay");
const moveCount = document.getElementById("moveCount");

function setStatus(text, cls) {
  statusEl.textContent = text;
  statusEl.className = cls || "";
}

function setGameId(id) {
  gameId = id;
  gameIdInput.value = id || "";
  history.replaceState(null, "", urlWithGameId(location.href, id));
  subscribeToGame(id);
}

function subscribeToGame(id) {
  if (closeSubscription) {
    closeSubscription();
    closeSubscription = null;
  }
  if (!id) return;
  closeSubscription = openGameSubscription(WebSocket, wsUrlFor(location.href), id, {
    onGame: (game) => {
      selected = null;
      draw(game);
    },
    onError: (e) => setStatus(e.message, "err"),
  });
}

function draw(game) {
  currentGame = game;
  renderBoard(document, boardEl, statusEl, game, selected, onTubeClick);
  renderMoveLog(document, moveLogEl, game.history || []);
  levelDisplay.textContent = game.levelId;
  difficultyDisplay.textContent = (levelsMap[game.levelId] || "—").toUpperCase();
  moveCount.textContent = game.moves;
}

async function onTubeClick(i) {
  if (!gameId || !currentGame) return;
  if (selected === null) {
    selected = i;
    draw(currentGame);
    return;
  }
  if (selected === i) {
    selected = null;
    draw(currentGame);
    return;
  }
  const from = selected;
  selected = null;
  try {
    const data = await gql(
      fetch,
      `mutation($g: ID!, $f: Int!, $t: Int!) { move(gameId: $g, from: $f, to: $t) { levelId capacity tubes moves solved stuck history { from to } } }`,
      { g: gameId, f: from + 1, t: i + 1 }
    );
    draw(data.move);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function loadLevels() {
  const data = await gql(fetch, `{ levels { id difficulty } }`, {});
  levelSelect.innerHTML = "";
  levelsMap = {};
  data.levels.forEach((lvl) => {
    levelsMap[lvl.id] = lvl.difficulty;
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
      fetch,
      `mutation($l: Int!) { newGame(levelId: $l) { id levelId capacity tubes moves solved stuck history { from to } } }`,
      { l: levelId }
    );
    setGameId(data.newGame.id);
    draw(data.newGame);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function resetGame() {
  if (!gameId) return newGame();
  selected = null;
  try {
    const data = await gql(
      fetch,
      `mutation($g: ID!) { resetGame(gameId: $g) { levelId capacity tubes moves solved stuck history { from to } } }`,
      { g: gameId }
    );
    draw(data.resetGame);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function undo() {
  if (!gameId) return;
  selected = null;
  try {
    const data = await gql(
      fetch,
      `mutation($g: ID!) { undo(gameId: $g) { levelId capacity tubes moves solved stuck history { from to } } }`,
      { g: gameId }
    );
    draw(data.undo);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function joinGame(id) {
  id = (id || gameIdInput.value).trim();
  if (!id) return;
  selected = null;
  try {
    const data = await gql(
      fetch,
      `query($g: ID!) { game(id: $g) { id levelId capacity tubes moves solved stuck history { from to } } }`,
      { g: id }
    );
    if (!data.game) throw new Error("game not found");
    setGameId(data.game.id);
    levelSelect.value = String(data.game.levelId);
    draw(data.game);
  } catch (e) {
    setStatus(e.message, "err");
  }
}

async function copyGameId() {
  if (!gameId) return;
  try {
    await navigator.clipboard.writeText(gameId);
    setStatus(`game id copied: ${gameId}`);
  } catch (e) {
    setStatus(gameId);
  }
}

document.getElementById("newGameBtn").addEventListener("click", newGame);
document.getElementById("resetBtn").addEventListener("click", resetGame);
document.getElementById("undoBtn").addEventListener("click", undo);
document.getElementById("joinBtn").addEventListener("click", () => joinGame());
document.getElementById("copyIdBtn").addEventListener("click", copyGameId);

const urlGameId = gameIdFromUrl(location.href);
loadLevels().then(() => (urlGameId ? joinGame(urlGameId) : newGame()));
