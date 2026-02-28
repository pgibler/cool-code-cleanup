const express = require("express");
const app = express();

app.post("/auth/login", (req, res) => {
  res.json({ token: "abc" });
});

app.get("/account/private", (req, res) => {
  res.json({ ok: true });
});

module.exports = app;
